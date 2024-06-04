package ledgerexporter

import (
	"context"
	_ "embed"
	"fmt"
	"os/exec"
	"strings"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"

	"github.com/pelletier/go-toml"

	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/ordered"
)

const Pubnet = "pubnet"
const Testnet = "testnet"

type Flags struct {
	StartLedger    uint32
	EndLedger      uint32
	ConfigFilePath string
	Resume         bool
	AdminPort      uint
}

type StellarCoreConfig struct {
	NetworkPassphrase     string   `toml:"network_passphrase"`
	HistoryArchiveUrls    []string `toml:"history_archive_urls"`
	StellarCoreBinaryPath string   `toml:"stellar_core_binary_path"`
	CaptiveCoreTomlPath   string   `toml:"captive_core_toml_path"`
}

type Config struct {
	AdminPort int `toml:"admin_port"`

	Network           string                      `toml:"network"`
	DataStoreConfig   datastore.DataStoreConfig   `toml:"datastore_config"`
	LedgerBatchConfig datastore.LedgerBatchConfig `toml:"exporter_config"`
	StellarCoreConfig StellarCoreConfig           `toml:"stellar_core_config"`

	StartLedger uint32
	EndLedger   uint32
	Resume      bool

	CoreVersion string
}

// This will generate the config based on commandline flags and toml
//
// flags                 - command line flags
//
// return                - *Config or an error if any range validation failed.
func NewConfig(flags Flags) (*Config, error) {
	config := &Config{}

	config.StartLedger = uint32(flags.StartLedger)
	config.EndLedger = uint32(flags.EndLedger)
	config.Resume = flags.Resume

	logger.Infof("Requested ledger range start=%d, end=%d, resume=%v", config.StartLedger, config.EndLedger, config.Resume)

	var err error
	if err = config.processToml(flags.ConfigFilePath); err != nil {
		return nil, err
	}
	logger.Infof("Config: %v", *config)

	return config, nil
}

// Validates requested ledger range, and will automatically adjust it
// to be ledgers-per-file boundary aligned
func (config *Config) ValidateAndSetLedgerRange(ctx context.Context, archive historyarchive.ArchiveInterface) error {
	latestNetworkLedger, err := datastore.GetLatestLedgerSequenceFromHistoryArchives(archive)

	if err != nil {
		return errors.Wrap(err, "Failed to retrieve the latest ledger sequence from history archives.")
	}
	logger.Infof("Latest %v ledger sequence was detected as %d", config.Network, latestNetworkLedger)

	if config.StartLedger > latestNetworkLedger {
		return errors.Errorf("--start %d exceeds latest network ledger %d",
			config.StartLedger, latestNetworkLedger)
	}

	if config.EndLedger != 0 { // Bounded mode
		if config.EndLedger < config.StartLedger {
			return errors.New("invalid --end value, must be >= --start")
		}
		if config.EndLedger > latestNetworkLedger {
			return errors.Errorf("--end %d exceeds latest network ledger %d",
				config.EndLedger, latestNetworkLedger)
		}
	}

	config.adjustLedgerRange()
	return nil
}

func (config *Config) generateCaptiveCoreConfig() (ledgerbackend.CaptiveCoreConfig, error) {
	coreConfig := &config.StellarCoreConfig

	// Look for stellar-core binary in $PATH, if not supplied
	if config.StellarCoreConfig.StellarCoreBinaryPath == "" {
		var err error
		if config.StellarCoreConfig.StellarCoreBinaryPath, err = exec.LookPath("stellar-core"); err != nil {
			return ledgerbackend.CaptiveCoreConfig{}, errors.Wrap(err, "Failed to find stellar-core binary")
		}
	}

	if err := config.setCoreVersionInfo(); err != nil {
		return ledgerbackend.CaptiveCoreConfig{}, fmt.Errorf("failed to set stellar-core version info: %w", err)
	}

	var captiveCoreConfig []byte
	// Default network config
	switch config.Network {
	case Pubnet:
		coreConfig.NetworkPassphrase = network.PublicNetworkPassphrase
		coreConfig.HistoryArchiveUrls = network.PublicNetworkhistoryArchiveURLs
		captiveCoreConfig = ledgerbackend.PubnetDefaultConfig

	case Testnet:
		coreConfig.NetworkPassphrase = network.TestNetworkPassphrase
		coreConfig.HistoryArchiveUrls = network.TestNetworkhistoryArchiveURLs
		captiveCoreConfig = ledgerbackend.TestnetDefaultConfig

	default:
		logger.Fatalf("Invalid network %s", config.Network)
	}

	params := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  coreConfig.NetworkPassphrase,
		HistoryArchiveURLs: coreConfig.HistoryArchiveUrls,
		UseDB:              true,
	}

	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromData(captiveCoreConfig, params)
	if err != nil {
		return ledgerbackend.CaptiveCoreConfig{}, errors.Wrap(err, "Failed to create captive-core toml")
	}

	return ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          coreConfig.StellarCoreBinaryPath,
		NetworkPassphrase:   params.NetworkPassphrase,
		HistoryArchiveURLs:  params.HistoryArchiveURLs,
		CheckpointFrequency: datastore.GetHistoryArchivesCheckPointFrequency(),
		Log:                 logger.WithField("subservice", "stellar-core"),
		Toml:                captiveCoreToml,
		UserAgent:           "ledger-exporter",
		UseDB:               true,
	}, nil
}

// By default, it points to exec.Command, overridden for testing purpose
var execCommand = exec.Command

// Executes the "stellar-core version" command and parses its output to extract
// the core version
// The output of the "version" command is expected to be a multi-line string where the
// first line is the core version in format "vX.Y.Z-*".
func (c *Config) setCoreVersionInfo() (err error) {
	versionCmd := execCommand(c.StellarCoreConfig.StellarCoreBinaryPath, "version")
	versionOutput, err := versionCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute stellar-core version command: %w", err)
	}

	// Split the output into lines
	rows := strings.Split(string(versionOutput), "\n")
	if len(rows) == 0 || len(rows[0]) == 0 {
		return fmt.Errorf("stellar-core version not found")
	}
	c.CoreVersion = rows[0]
	logger.Infof("stellar-core version: %s", c.CoreVersion)
	return nil
}

func (config *Config) processToml(tomlPath string) error {
	// Load config TOML file
	cfg, err := toml.LoadFile(tomlPath)
	if err != nil {
		return err
	}

	// Unmarshal TOML data into the Config struct
	if err := cfg.Unmarshal(config); err != nil {
		return errors.Wrap(err, "Error unmarshalling TOML config.")
	}

	// validate TOML data
	if config.Network == "" {
		return errors.New("Invalid TOML config, 'network' must be set, supported values are 'testnet' or 'pubnet'")
	}
	return nil
}

func (config *Config) adjustLedgerRange() {

	// Check if either the start or end ledger does not fall on the "LedgersPerFile" boundary
	// and adjust the start and end ledger accordingly.
	// Align the start ledger to the nearest "LedgersPerFile" boundary.
	config.StartLedger = config.LedgerBatchConfig.GetSequenceNumberStartBoundary(config.StartLedger)

	// Ensure that the adjusted start ledger is at least 2.
	config.StartLedger = ordered.Max(2, config.StartLedger)

	// Align the end ledger (for bounded cases) to the nearest "LedgersPerFile" boundary.
	if config.EndLedger != 0 {
		config.EndLedger = config.LedgerBatchConfig.GetSequenceNumberEndBoundary(config.EndLedger)
	}

	logger.Infof("Computed effective export boundary ledger range: start=%d, end=%d", config.StartLedger, config.EndLedger)
}
