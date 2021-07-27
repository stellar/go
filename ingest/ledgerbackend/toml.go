package ledgerbackend

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"

	"github.com/pelletier/go-toml"
)

const (
	defaultHTTPPort      = 11626
	defaultFailureSafety = -1

	// if LOG_FILE_PATH is omitted stellar core actually defaults to "stellar-core.log"
	// however, we are overriding this default for captive core
	defaultLogFilePath = "" // by default we disable logging to a file
)

var validQuality = map[string]bool{
	"CRITICAL": true,
	"HIGH":     true,
	"MEDIUM":   true,
	"LOW":      true,
}

// Validator represents a [[VALIDATORS]] entry in the captive core toml file.
type Validator struct {
	Name       string `toml:"NAME"`
	Quality    string `toml:"QUALITY,omitempty"`
	HomeDomain string `toml:"HOME_DOMAIN"`
	PublicKey  string `toml:"PUBLIC_KEY"`
	Address    string `toml:"ADDRESS,omitempty"`
	History    string `toml:"HISTORY,omitempty"`
}

// HomeDomain represents a [[HOME_DOMAINS]] entry in the captive core toml file.
type HomeDomain struct {
	HomeDomain string `toml:"HOME_DOMAIN"`
	Quality    string `toml:"QUALITY"`
}

// History represents a [HISTORY] table in the captive core toml file.
type History struct {
	Get string `toml:"get"`
	// should we allow put and mkdir for captive core?
	Put   string `toml:"put,omitempty"`
	Mkdir string `toml:"mkdir,omitempty"`
}

// QuorumSet represents a [QUORUM_SET] table in the captive core toml file.
type QuorumSet struct {
	ThresholdPercent int      `toml:"THRESHOLD_PERCENT"`
	Validators       []string `toml:"VALIDATORS"`
}

type captiveCoreTomlValues struct {
	// we cannot omitempty because the empty string is a valid configuration for LOG_FILE_PATH
	// and the default is stellar-core.log
	LogFilePath   string `toml:"LOG_FILE_PATH"`
	BucketDirPath string `toml:"BUCKET_DIR_PATH,omitempty"`
	// we cannot omitempty because 0 is a valid configuration for HTTP_PORT
	// and the default is 11626
	HTTPPort          uint     `toml:"HTTP_PORT"`
	PublicHTTPPort    bool     `toml:"PUBLIC_HTTP_PORT,omitempty"`
	NodeNames         []string `toml:"NODE_NAMES,omitempty"`
	NetworkPassphrase string   `toml:"NETWORK_PASSPHRASE,omitempty"`
	PeerPort          uint     `toml:"PEER_PORT,omitempty"`
	// we cannot omitempty because 0 is a valid configuration for FAILURE_SAFETY
	// and the default is -1
	FailureSafety                        int                  `toml:"FAILURE_SAFETY"`
	UnsafeQuorum                         bool                 `toml:"UNSAFE_QUORUM,omitempty"`
	RunStandalone                        bool                 `toml:"RUN_STANDALONE,omitempty"`
	ArtificiallyAccelerateTimeForTesting bool                 `toml:"ARTIFICIALLY_ACCELERATE_TIME_FOR_TESTING,omitempty"`
	HomeDomains                          []HomeDomain         `toml:"HOME_DOMAINS,omitempty"`
	Validators                           []Validator          `toml:"VALIDATORS,omitempty"`
	HistoryEntries                       map[string]History   `toml:"-"`
	QuorumSetEntries                     map[string]QuorumSet `toml:"-"`
}

// QuorumSetIsConfigured returns true if there is a quorum set defined in the configuration.
func (c *captiveCoreTomlValues) QuorumSetIsConfigured() bool {
	return len(c.QuorumSetEntries) > 0 || len(c.Validators) > 0
}

// HistoryIsConfigured returns true if the history archive locations are configured.
func (c *captiveCoreTomlValues) HistoryIsConfigured() bool {
	if len(c.HistoryEntries) > 0 {
		return true
	}
	for _, v := range c.Validators {
		if v.History != "" {
			return true
		}
	}
	return false
}

type placeholders struct {
	labels map[string]string
	count  int
}

func (p *placeholders) newPlaceholder(key string) string {
	if p.labels == nil {
		p.labels = map[string]string{}
	}
	placeHolder := fmt.Sprintf("__placeholder_label_%d__", p.count)
	p.count++
	p.labels[placeHolder] = key
	return placeHolder
}

func (p *placeholders) get(placeholder string) (string, bool) {
	if p.labels == nil {
		return "", false
	}
	val, ok := p.labels[placeholder]
	return val, ok
}

// CaptiveCoreToml represents a parsed captive core configuration.
type CaptiveCoreToml struct {
	captiveCoreTomlValues
	tree              *toml.Tree
	tablePlaceholders *placeholders
}

// flattenTables will transform a given toml text by flattening all nested tables
// whose root can be found in `rootNames`.
//
// In the TOML spec dotted keys represents nesting. So we flatten the table key by replacing each table
// path with a placeholder. For example:
//
// text := `[QUORUM_SET.a.b.c]
//         THRESHOLD_PERCENT=67
//         VALIDATORS=["a","b"]`
// flattenTables(text, []string{"QUORUM_SET"}) ->
//
// `[__placeholder_label_0__]
// THRESHOLD_PERCENT=67
// VALIDATORS=["a","b"]`
func flattenTables(text string, rootNames []string) (string, *placeholders) {
	orExpression := strings.Join(rootNames, "|")
	re := regexp.MustCompile(`\[(` + orExpression + `)(\..+)?\]`)

	tablePlaceHolders := &placeholders{}

	flattened := re.ReplaceAllStringFunc(text, func(match string) string {
		insideBrackets := match[1 : len(match)-1]
		return "[" + tablePlaceHolders.newPlaceholder(insideBrackets) + "]"
	})
	return flattened, tablePlaceHolders
}

// unflattenTables is the inverse of flattenTables, it restores the
// text back to its original form by replacing all placeholders with their
// original values.
func unflattenTables(text string, tablePlaceHolders *placeholders) string {
	re := regexp.MustCompile(`\[.*\]`)

	return re.ReplaceAllStringFunc(text, func(match string) string {
		insideBrackets := match[1 : len(match)-1]
		original, ok := tablePlaceHolders.get(insideBrackets)
		if !ok {
			return match
		}
		return "[" + original + "]"
	})
}

// AddExamplePubnetQuorum adds example pubnet validators to toml file
func (c *CaptiveCoreToml) AddExamplePubnetValidators() {
	c.captiveCoreTomlValues.Validators = []Validator{
		{
			Name:       "sdf_1",
			HomeDomain: "stellar.org",
			PublicKey:  "GCGB2S2KGYARPVIA37HYZXVRM2YZUEXA6S33ZU5BUDC6THSB62LZSTYH",
			Address:    "core-live-a.stellar.org:11625",
			History:    "curl -sf https://history.stellar.org/prd/core-live/core_live_001/{0} -o {1}",
		},
		{
			Name:       "sdf_2",
			HomeDomain: "stellar.org",
			PublicKey:  "GCM6QMP3DLRPTAZW2UZPCPX2LF3SXWXKPMP3GKFZBDSF3QZGV2G5QSTK",
			Address:    "core-live-b.stellar.org:11625",
			History:    "curl -sf https://history.stellar.org/prd/core-live/core_live_002/{0} -o {1}",
		},
		{
			Name:       "sdf_3",
			HomeDomain: "stellar.org",
			PublicKey:  "GABMKJM6I25XI4K7U6XWMULOUQIQ27BCTMLS6BYYSOWKTBUXVRJSXHYQ",
			Address:    "core-live-c.stellar.org:11625",
			History:    "curl -sf https://history.stellar.org/prd/core-live/core_live_003/{0} -o {1}",
		},
	}
}

// Marshal serializes the CaptiveCoreToml into a toml document.
func (c *CaptiveCoreToml) Marshal() ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("# Generated file, do not edit\n")
	encoder := toml.NewEncoder(&sb)
	if err := encoder.Encode(c.captiveCoreTomlValues); err != nil {
		return nil, errors.Wrap(err, "could not encode toml file")
	}

	if len(c.HistoryEntries) > 0 {
		if err := encoder.Encode(c.HistoryEntries); err != nil {
			return nil, errors.Wrap(err, "could not encode history entries")
		}
	}

	if len(c.QuorumSetEntries) > 0 {
		if err := encoder.Encode(c.QuorumSetEntries); err != nil {
			return nil, errors.Wrap(err, "could not encode quorum set")
		}
	}

	return []byte(unflattenTables(sb.String(), c.tablePlaceholders)), nil
}

func unmarshalTreeNode(t *toml.Tree, key string, dest interface{}) error {
	tree, ok := t.Get(key).(*toml.Tree)
	if !ok {
		return fmt.Errorf("unexpected key %v", key)
	}
	return tree.Unmarshal(dest)
}

func (c *CaptiveCoreToml) unmarshal(data []byte, strict bool) error {
	quorumSetEntries := map[string]QuorumSet{}
	historyEntries := map[string]History{}
	// The toml library has trouble with nested tables so we need to flatten all nested
	// QUORUM_SET and HISTORY tables as a workaround.
	// In Marshal() we apply the inverse process to unflatten the nested tables.
	flattened, tablePlaceholders := flattenTables(string(data), []string{"QUORUM_SET", "HISTORY"})

	tree, err := toml.Load(flattened)
	if err != nil {
		return err
	}

	for _, key := range tree.Keys() {
		originalKey, ok := tablePlaceholders.get(key)
		if !ok {
			continue
		}

		switch {
		case strings.HasPrefix(originalKey, "QUORUM_SET"):
			var qs QuorumSet
			if err = unmarshalTreeNode(tree, key, &qs); err != nil {
				return err
			}
			quorumSetEntries[key] = qs
		case strings.HasPrefix(originalKey, "HISTORY"):
			var h History
			if err = unmarshalTreeNode(tree, key, &h); err != nil {
				return err
			}
			historyEntries[key] = h
		}
		if err = tree.Delete(key); err != nil {
			return err
		}
	}

	var body captiveCoreTomlValues
	if withoutPlaceHolders, err := tree.Marshal(); err != nil {
		return err
	} else if err = toml.NewDecoder(bytes.NewReader(withoutPlaceHolders)).Strict(strict).Decode(&body); err != nil {
		if message := err.Error(); strings.HasPrefix(message, "undecoded keys") {
			return fmt.Errorf(strings.Replace(
				message,
				"undecoded keys",
				"these fields are not supported by captive core",
				1,
			))
		}
		return err
	}

	c.tree = tree
	c.captiveCoreTomlValues = body
	c.tablePlaceholders = tablePlaceholders
	c.QuorumSetEntries = quorumSetEntries
	c.HistoryEntries = historyEntries
	return nil
}

// CaptiveCoreTomlParams defines captive core configuration provided by Horizon flags.
type CaptiveCoreTomlParams struct {
	// NetworkPassphrase is the Stellar network passphrase used by captive core when connecting to the Stellar network.
	NetworkPassphrase string
	// HistoryArchiveURLs are a list of history archive urls.
	HistoryArchiveURLs []string
	// HTTPPort is the TCP port to listen for requests (defaults to 0, which disables the HTTP server).
	HTTPPort *uint
	// PeerPort is the TCP port to bind to for connecting to the Stellar network
	// (defaults to 11625). It may be useful for example when there's >1 Stellar-Core
	// instance running on a machine.
	PeerPort *uint
	// LogPath is the (optional) path in which to store Core logs, passed along
	// to Stellar Core's LOG_FILE_PATH.
	LogPath *string
	// Strict is a flag which, if enabled, rejects Stellar Core toml fields which are not supported by captive core.
	Strict bool
}

// NewCaptiveCoreTomlFromFile constructs a new CaptiveCoreToml instance by merging configuration
// from the toml file located at `configPath` and the configuration provided by `params`.
func NewCaptiveCoreTomlFromFile(configPath string, params CaptiveCoreTomlParams) (*CaptiveCoreToml, error) {
	var captiveCoreToml CaptiveCoreToml
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not load toml path")
	}

	if err = captiveCoreToml.unmarshal(data, params.Strict); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal captive core toml")
	}
	// disallow setting BUCKET_DIR_PATH through a file since it can cause multiple
	// running captive-core instances to clash
	if params.Strict && captiveCoreToml.BucketDirPath != "" {
		return nil, errors.New("could not unmarshal captive core toml: setting BUCKET_DIR_PATH is disallowed, it can cause clashes between instances")
	}

	if err = captiveCoreToml.validate(params); err != nil {
		return nil, errors.Wrap(err, "invalid captive core toml")
	}

	if len(captiveCoreToml.HistoryEntries) > 0 {
		log.Warnf(
			"Configuring captive core with history archive from %s instead of %v",
			configPath,
			params.HistoryArchiveURLs,
		)
	}

	captiveCoreToml.setDefaults(params)
	return &captiveCoreToml, nil
}

// NewDefaultTestnetCaptiveCoreToml constructs a new CaptiveCoreToml instance
// based off the default testnet configuration.
func NewDefaultTestnetCaptiveCoreToml() *CaptiveCoreToml {
	var captiveCoreToml CaptiveCoreToml

	captiveCoreToml.tablePlaceholders = &placeholders{}

	captiveCoreToml.PublicHTTPPort = true
	captiveCoreToml.HTTPPort = 11626

	captiveCoreToml.FailureSafety = -1
	captiveCoreToml.NetworkPassphrase = network.TestNetworkPassphrase

	captiveCoreToml.HomeDomains = []HomeDomain{
		{
			HomeDomain: "testnet.stellar.org",
			Quality:    "LOW",
		},
	}

	captiveCoreToml.Validators = []Validator{
		{
			Name:       "sdf_testnet_1",
			HomeDomain: "testnet.stellar.org",
			PublicKey:  "GDKXE2OZMJIPOSLNA6N6F2BVCI3O777I2OOC4BV7VOYUEHYX7RTRYA7Y",
			Address:    "core-testnet1.stellar.org",
			History:    "curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_001/{0} -o {1}",
		},
		{
			Name:       "sdf_testnet_2",
			HomeDomain: "testnet.stellar.org",
			PublicKey:  "GCUCJTIYXSOXKBSNFGNFWW5MUQ54HKRPGJUTQFJ5RQXZXNOLNXYDHRAP",
			Address:    "core-testnet2.stellar.org",
			History:    "curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_002/{0} -o {1}",
		},
		{
			Name:       "sdf_testnet_3",
			HomeDomain: "testnet.stellar.org",
			PublicKey:  "GC2V2EFSXN6SQTWVYA5EPJPBWWIMSD2XQNKUOHGEKB535AQE2I6IXV2Z",
			Address:    "core-testnet3.stellar.org",
			History:    "curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_003/{0} -o {1}",
		},
	}

	return &captiveCoreToml
}

// NewDefaultPubnetCaptiveCoreToml constructs a new CaptiveCoreToml instance
// based off the default pubnet configuration.
func NewDefaultPubnetCaptiveCoreToml() *CaptiveCoreToml {
	var captiveCoreToml CaptiveCoreToml

	captiveCoreToml.tablePlaceholders = &placeholders{}

	captiveCoreToml.PublicHTTPPort = true
	captiveCoreToml.HTTPPort = 11626

	captiveCoreToml.FailureSafety = -1
	captiveCoreToml.NetworkPassphrase = network.PublicNetworkPassphrase

	captiveCoreToml.HomeDomains = []HomeDomain{
		{
			HomeDomain: "stellar.org",
			Quality:    "HIGH",
		},
		{
			HomeDomain: "satoshipay.io",
			Quality:    "HIGH",
		},
		{
			HomeDomain: "lobstr.co",
			Quality:    "HIGH",
		},
		{
			HomeDomain: "www.coinqvest.com",
			Quality:    "HIGH",
		},
		{
			HomeDomain: "keybase.io",
			Quality:    "HIGH",
		},
		{
			HomeDomain: "stellar.blockdaemon.com",
			Quality:    "HIGH",
		},
		{
			HomeDomain: "wirexapp.com",
			Quality:    "HIGH",
		},
	}

	captiveCoreToml.Validators = []Validator{
		{
			Name:       "sdf_1",
			HomeDomain: "stellar.org",
			PublicKey:  "GCGB2S2KGYARPVIA37HYZXVRM2YZUEXA6S33ZU5BUDC6THSB62LZSTYH",
			Address:    "core-live-a.stellar.org:11625",
			History:    "curl -sf https://history.stellar.org/prd/core-live/core_live_001/{0} -o {1}",
		},
		{
			Name:       "sdf_2",
			HomeDomain: "stellar.org",
			PublicKey:  "GCM6QMP3DLRPTAZW2UZPCPX2LF3SXWXKPMP3GKFZBDSF3QZGV2G5QSTK",
			Address:    "core-live-b.stellar.org:11625",
			History:    "curl -sf https://history.stellar.org/prd/core-live/core_live_002/{0} -o {1}",
		},
		{
			Name:       "sdf_3",
			HomeDomain: "stellar.org",
			PublicKey:  "GABMKJM6I25XI4K7U6XWMULOUQIQ27BCTMLS6BYYSOWKTBUXVRJSXHYQ",
			Address:    "core-live-c.stellar.org:11625",
			History:    "curl -sf https://history.stellar.org/prd/core-live/core_live_003/{0} -o {1}",
		},
		{
			Name:       "satoshipay_singapore",
			HomeDomain: "satoshipay.io",
			PublicKey:  "GBJQUIXUO4XSNPAUT6ODLZUJRV2NPXYASKUBY4G5MYP3M47PCVI55MNT",
			Address:    "stellar-sg-sin.satoshipay.io:11625",
			History:    "curl -sf https://stellar-history-sg-sin.satoshipay.io/{0} -o {1}",
		},
		{
			Name:       "satoshipay_iowa",
			HomeDomain: "satoshipay.io",
			PublicKey:  "GAK6Z5UVGUVSEK6PEOCAYJISTT5EJBB34PN3NOLEQG2SUKXRVV2F6HZY",
			Address:    "stellar-us-iowa.satoshipay.io:11625",
			History:    "curl -sf https://stellar-history-us-iowa.satoshipay.io/{0} -o {1}",
		},
		{
			Name:       "satoshipay_frankfurt",
			HomeDomain: "satoshipay.io",
			PublicKey:  "GC5SXLNAM3C4NMGK2PXK4R34B5GNZ47FYQ24ZIBFDFOCU6D4KBN4POAE",
			Address:    "stellar-de-fra.satoshipay.io:11625",
			History:    "curl -sf https://stellar-history-de-fra.satoshipay.io/{0} -o {1}",
		},
		{
			Name:       "lobstr_1_europe",
			HomeDomain: "lobstr.co",
			PublicKey:  "GCFONE23AB7Y6C5YZOMKUKGETPIAJA4QOYLS5VNS4JHBGKRZCPYHDLW7",
			Address:    "v1.stellar.lobstr.co:11625",
			History:    "curl -sf https://stellar-archive-1-lobstr.s3.amazonaws.com/{0} -o {1}",
		},
		{
			Name:       "lobstr_2_europe",
			HomeDomain: "lobstr.co",
			PublicKey:  "GDXQB3OMMQ6MGG43PWFBZWBFKBBDUZIVSUDAZZTRAWQZKES2CDSE5HKJ",
			Address:    "v2.stellar.lobstr.co:11625",
			History:    "curl -sf https://stellar-archive-2-lobstr.s3.amazonaws.com/{0} -o {1}",
		},
		{
			Name:       "lobstr_3_north_america",
			HomeDomain: "lobstr.co",
			PublicKey:  "GD5QWEVV4GZZTQP46BRXV5CUMMMLP4JTGFD7FWYJJWRL54CELY6JGQ63",
			Address:    "v3.stellar.lobstr.co:11625",
			History:    "curl -sf https://stellar-archive-3-lobstr.s3.amazonaws.com/{0} -o {1}",
		},
		{
			Name:       "lobstr_4_asia",
			HomeDomain: "lobstr.co",
			PublicKey:  "GA7TEPCBDQKI7JQLQ34ZURRMK44DVYCIGVXQQWNSWAEQR6KB4FMCBT7J",
			Address:    "v4.stellar.lobstr.co:11625",
			History:    "curl -sf https://stellar-archive-4-lobstr.s3.amazonaws.com/{0} -o {1}",
		},
		{
			Name:       "lobstr_5_australia",
			HomeDomain: "lobstr.co",
			PublicKey:  "GA5STBMV6QDXFDGD62MEHLLHZTPDI77U3PFOD2SELU5RJDHQWBR5NNK7",
			Address:    "v5.stellar.lobstr.co:11625",
			History:    "curl -sf https://stellar-archive-5-lobstr.s3.amazonaws.com/{0} -o {1}",
		},
		{
			Name:       "coinqvest_hong_kong",
			HomeDomain: "www.coinqvest.com",
			PublicKey:  "GAZ437J46SCFPZEDLVGDMKZPLFO77XJ4QVAURSJVRZK2T5S7XUFHXI2Z",
			Address:    "hongkong.stellar.coinqvest.com:11625",
			History:    "curl -sf https://hongkong.stellar.coinqvest.com/history/{0} -o {1}",
		},
		{
			Name:       "coinqvest_germany",
			HomeDomain: "www.coinqvest.com",
			PublicKey:  "GD6SZQV3WEJUH352NTVLKEV2JM2RH266VPEM7EH5QLLI7ZZAALMLNUVN",
			Address:    "germany.stellar.coinqvest.com:11625",
			History:    "curl -sf https://germany.stellar.coinqvest.com/history/{0} -o {1}",
		},
		{
			Name:       "coinqvest_finland",
			HomeDomain: "www.coinqvest.com",
			PublicKey:  "GADLA6BJK6VK33EM2IDQM37L5KGVCY5MSHSHVJA4SCNGNUIEOTCR6J5T",
			Address:    "finland.stellar.coinqvest.com:11625",
			History:    "curl -sf https://finland.stellar.coinqvest.com/history/{0} -o {1}",
		},
		{
			Name:       "keybase_io",
			HomeDomain: "keybase.io",
			PublicKey:  "GCWJKM4EGTGJUVSWUJDPCQEOEP5LHSOFKSA4HALBTOO4T4H3HCHOM6UX",
			Address:    "stellar0.keybase.io:11625",
			History:    "curl -sf https://stellarhistory.keybase.io/{0} -o {1}",
		},
		{
			Name:       "keybase_1",
			HomeDomain: "keybase.io",
			PublicKey:  "GDKWELGJURRKXECG3HHFHXMRX64YWQPUHKCVRESOX3E5PM6DM4YXLZJM",
			Address:    "stellar1.keybase.io:11625",
			History:    "curl -sf https://stellarhistory1.keybase.io/{0} -o {1}",
		},
		{
			Name:       "keybase_2",
			HomeDomain: "keybase.io",
			PublicKey:  "GA35T3723UP2XJLC2H7MNL6VMKZZIFL2VW7XHMFFJKKIA2FJCYTLKFBW",
			Address:    "stellar2.keybase.io:11625",
			History:    "curl -sf https://stellarhistory2.keybase.io/{0} -o {1}",
		},
		{
			Name:       "Blockdaemon_Validator_1",
			HomeDomain: "stellar.blockdaemon.com",
			PublicKey:  "GAAV2GCVFLNN522ORUYFV33E76VPC22E72S75AQ6MBR5V45Z5DWVPWEU",
			Address:    "stellar-full-validator1.bdnodes.net",
			History:    "curl -sf https://stellar-full-history1.bdnodes.net/{0} -o {1}",
		},
		{
			Name:       "Blockdaemon_Validator_2",
			HomeDomain: "stellar.blockdaemon.com",
			PublicKey:  "GAVXB7SBJRYHSG6KSQHY74N7JAFRL4PFVZCNWW2ARI6ZEKNBJSMSKW7C",
			Address:    "stellar-full-validator2.bdnodes.net",
			History:    "curl -sf https://stellar-full-history2.bdnodes.net/{0} -o {1}",
		},
		{
			Name:       "Blockdaemon_Validator_3",
			HomeDomain: "stellar.blockdaemon.com",
			PublicKey:  "GAYXZ4PZ7P6QOX7EBHPIZXNWY4KCOBYWJCA4WKWRKC7XIUS3UJPT6EZ4",
			Address:    "stellar-full-validator3.bdnodes.net",
			History:    "curl -sf https://stellar-full-history3.bdnodes.net/{0} -o {1}",
		},
		{
			Name:       "wirexUS",
			HomeDomain: "wirexapp.com",
			PublicKey:  "GDXUKFGG76WJC7ACEH3JUPLKM5N5S76QSMNDBONREUXPCZYVPOLFWXUS",
			Address:    "us.stellar.wirexapp.com",
			History:    "curl -sf http://wxhorizonusstga1.blob.core.windows.net/history/{0} -o {1}",
		},
		{
			Name:       "wirexUK",
			HomeDomain: "wirexapp.com",
			PublicKey:  "GBBQQT3EIUSXRJC6TGUCGVA3FVPXVZLGG3OJYACWBEWYBHU46WJLWXEU",
			Address:    "uk.stellar.wirexapp.com",
			History:    "curl -sf http://wxhorizonukstga1.blob.core.windows.net/history/{0} -o {1}",
		},
		{
			Name:       "wirexSG",
			HomeDomain: "wirexapp.com",
			PublicKey:  "GAB3GZIE6XAYWXGZUDM4GMFFLJBFMLE2JDPUCWUZXMOMT3NHXDHEWXAS",
			Address:    "sg.stellar.wirexapp.com",
			History:    "curl -sf http://wxhorizonasiastga1.blob.core.windows.net/history/{0} -o {1}",
		},
	}

	return &captiveCoreToml
}

// NewCaptiveCoreToml constructs a new CaptiveCoreToml instance based off
// the configuration in `params`.
func NewCaptiveCoreToml(params CaptiveCoreTomlParams) (*CaptiveCoreToml, error) {
	var captiveCoreToml CaptiveCoreToml
	var err error

	captiveCoreToml.tablePlaceholders = &placeholders{}
	captiveCoreToml.tree, err = toml.TreeFromMap(map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	captiveCoreToml.setDefaults(params)
	return &captiveCoreToml, nil
}

func (c *CaptiveCoreToml) clone() (*CaptiveCoreToml, error) {
	data, err := c.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal toml")
	}
	var cloned CaptiveCoreToml
	if err = cloned.unmarshal(data, false); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal captive core toml")
	}
	return &cloned, nil
}

// CatchupToml returns a new CaptiveCoreToml instance based off the existing
// instance with some modifications which are suitable for running
// the catchup command on captive core.
func (c *CaptiveCoreToml) CatchupToml() (*CaptiveCoreToml, error) {
	offline, err := c.clone()
	if err != nil {
		return nil, errors.Wrap(err, "could not clone toml")
	}

	offline.RunStandalone = true
	offline.UnsafeQuorum = true
	offline.PublicHTTPPort = false
	offline.HTTPPort = 0
	offline.FailureSafety = 0

	if !c.QuorumSetIsConfigured() {
		// Add a fictional quorum -- necessary to convince core to start up;
		// but not used at all for our purposes. Pubkey here is just random.
		offline.QuorumSetEntries = map[string]QuorumSet{
			"QUORUM_SET": {
				ThresholdPercent: 100,
				Validators:       []string{"GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"},
			},
		}
	}
	return offline, nil
}

func (c *CaptiveCoreToml) setDefaults(params CaptiveCoreTomlParams) {
	if !c.tree.Has("NETWORK_PASSPHRASE") {
		c.NetworkPassphrase = params.NetworkPassphrase
	}

	if def := c.tree.Has("HTTP_PORT"); !def && params.HTTPPort != nil {
		c.HTTPPort = *params.HTTPPort
	} else if !def && params.HTTPPort == nil {
		c.HTTPPort = defaultHTTPPort
	}

	if def := c.tree.Has("PEER_PORT"); !def && params.PeerPort != nil {
		c.PeerPort = *params.PeerPort
	}

	if def := c.tree.Has("LOG_FILE_PATH"); !def && params.LogPath != nil {
		c.LogFilePath = *params.LogPath
	} else if !def && params.LogPath == nil {
		c.LogFilePath = defaultLogFilePath
	}

	if !c.tree.Has("FAILURE_SAFETY") {
		c.FailureSafety = defaultFailureSafety
	}
	if !c.HistoryIsConfigured() {
		c.HistoryEntries = map[string]History{}
		for i, val := range params.HistoryArchiveURLs {
			name := fmt.Sprintf("HISTORY.h%d", i)
			c.HistoryEntries[c.tablePlaceholders.newPlaceholder(name)] = History{
				Get: fmt.Sprintf("curl -sf %s/{0} -o {1}", val),
			}
		}
	}
}

func (c *CaptiveCoreToml) validate(params CaptiveCoreTomlParams) error {
	if def := c.tree.Has("NETWORK_PASSPHRASE"); def && c.NetworkPassphrase != params.NetworkPassphrase {
		return fmt.Errorf(
			"NETWORK_PASSPHRASE in captive core config file: %s does not match Horizon network-passphrase flag: %s",
			c.NetworkPassphrase,
			params.NetworkPassphrase,
		)
	}

	if def := c.tree.Has("HTTP_PORT"); def && params.HTTPPort != nil && c.HTTPPort != *params.HTTPPort {
		return fmt.Errorf(
			"HTTP_PORT in captive core config file: %d does not match Horizon captive-core-http-port flag: %d",
			c.HTTPPort,
			*params.HTTPPort,
		)
	}

	if def := c.tree.Has("PEER_PORT"); def && params.PeerPort != nil && c.PeerPort != *params.PeerPort {
		return fmt.Errorf(
			"PEER_PORT in captive core config file: %d does not match Horizon captive-core-peer-port flag: %d",
			c.PeerPort,
			*params.PeerPort,
		)
	}

	if def := c.tree.Has("LOG_FILE_PATH"); def && params.LogPath != nil && c.LogFilePath != *params.LogPath {
		return fmt.Errorf(
			"LOG_FILE_PATH in captive core config file: %s does not match Horizon captive-core-log-path flag: %s",
			c.LogFilePath,
			*params.LogPath,
		)
	}

	homeDomainSet := map[string]HomeDomain{}
	for _, hd := range c.HomeDomains {
		if _, ok := homeDomainSet[hd.HomeDomain]; ok {
			return fmt.Errorf(
				"found duplicate home domain in captive core configuration: %s",
				hd.HomeDomain,
			)
		}
		if hd.HomeDomain == "" {
			return fmt.Errorf(
				"found invalid home domain entry which is missing a HOME_DOMAIN value",
			)
		}
		if hd.Quality == "" {
			return fmt.Errorf(
				"found invalid home domain entry which is missing a QUALITY value: %s",
				hd.HomeDomain,
			)
		}
		if !validQuality[hd.Quality] {
			return fmt.Errorf(
				"found invalid home domain entry which has an invalid QUALITY value: %s",
				hd.HomeDomain,
			)
		}
		homeDomainSet[hd.HomeDomain] = hd
	}

	names := map[string]bool{}
	for _, v := range c.Validators {
		if names[v.Name] {
			return fmt.Errorf(
				"found duplicate validator in captive core configuration: %s",
				v.Name,
			)
		}
		if v.Name == "" {
			return fmt.Errorf(
				"found invalid validator entry which is missing a NAME value: %s",
				v.Name,
			)
		}
		if v.HomeDomain == "" {
			return fmt.Errorf(
				"found invalid validator entry which is missing a HOME_DOMAIN value: %s",
				v.Name,
			)
		}
		if v.PublicKey == "" {
			return fmt.Errorf(
				"found invalid validator entry which is missing a PUBLIC_KEY value: %s",
				v.Name,
			)
		}
		if _, err := xdr.AddressToAccountId(v.PublicKey); err != nil {
			return fmt.Errorf(
				"found invalid validator entry which has an invalid PUBLIC_KEY : %s",
				v.Name,
			)
		}
		if v.Quality == "" {
			if _, ok := homeDomainSet[v.HomeDomain]; !ok {
				return fmt.Errorf(
					"found invalid validator entry which is missing a QUALITY value: %s",
					v.Name,
				)
			}
		} else if !validQuality[v.Quality] {
			return fmt.Errorf(
				"found invalid validator entry which has an invalid QUALITY value: %s",
				v.Name,
			)
		}

		names[v.Name] = true
	}

	return nil
}
