package ledgerbackend

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// Ensure captiveStellarCore implements LedgerBackend
var _ LedgerBackend = (*captiveStellarCore)(nil)

// This is a not-very-complete or well-organized sketch of code be used to
// stream LedgerCloseMeta data from a "captive" stellar-core: one running as a
// subprocess and replaying portions of history against an in-memory ledger.
//
// A captive stellar-core still needs (and allocates, in os.TempDir()) a
// temporary directory to run in: one in which its config file is stored, along
// with temporary files it downloads and decompresses, and its bucket
// state. Only the ledger will be in-memory (and we might even switch this to
// SQLite + large buffers in the future if the in-memory ledger gets too big.)
//
// Feel free to reorganize this to fit better. It's preliminary!

// TODO: switch from history URLs to history archive interface provided from support package, to permit mocking

// In this (crude, initial) sketch, we replay ledgers in blocks of 17,280
// which is 24 hours worth of ledgers at 5 second intervals.
const ledgersPerProcess = 17280
const ledgersPerCheckpoint = 64

// The number of checkpoints we're willing to scan over and ignore, without
// restarting a subprocess.
const numCheckpointsLeeway = 10

func roundDownToFirstReplayAfterCheckpointStart(ledger uint32) uint32 {
	v := (ledger / ledgersPerCheckpoint) * ledgersPerCheckpoint
	if v == 0 {
		return 1
	}
	// All other checkpoints start at the next multiple of 64
	return v
}

type captiveStellarCore struct {
	nonce             string
	networkPassphrase string
	historyURLs       []string
	lastLedger        *uint32 // end of current segment if offline, nil if online
	cmd               *exec.Cmd
	metaPipe          io.Reader

	nextLedgerMutex sync.Mutex
	nextLedger      uint32 // next ledger expected, error w/ restart if not seen
}

// NewCaptive returns a new captiveStellarCore that is not running. Will lazily start a subprocess
// to feed it a block of streaming metadata when user calls .GetLedger(), and will kill
// and restart the subprocess if subsequent calls to .GetLedger() are discontiguous.
//
// Platform-specific pipe setup logic is in the .start() methods.
func NewCaptive(networkPassphrase string, historyURLs []string) *captiveStellarCore {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &captiveStellarCore{
		nonce:             fmt.Sprintf("captive-stellar-core-%x", r.Uint64()),
		networkPassphrase: networkPassphrase,
		historyURLs:       historyURLs,
		nextLedger:        0,
	}
}

// Each captiveStellarCore is either doing bulk offline replay or tracking
// a network as it closes ledgers online. These cases are differentiated
// by the lastLedger field of captiveStellarCore, which is nil in the online
// case (indicating there's no end to the subprocess) and non-nil in the
// offline case (indicating that the subprocess will be closed after it yields
// the last ledger in the segment).
func (c *captiveStellarCore) IsInOfflineReplayMode() bool {
	return c.lastLedger != nil
}

func (c *captiveStellarCore) IsInOnlineTrackingMode() bool {
	return c.lastLedger == nil
}

// XDR and RPC define a (minimal) framing format which our metadata arrives in: a 4-byte
// big-endian length header that has the high bit set, followed by that length worth of
// XDR data. Decoding this involves just a little more work than xdr.Unmarshal.
func unmarshalFramed(r io.Reader, v interface{}) (int, error) {
	var frameLen uint32
	n, e := xdr.Unmarshal(r, &frameLen)
	if e != nil {
		err := errors.Wrap(e, "unmarshalling XDR frame header")
		return n, err
	}
	if n != 4 {
		err := errors.New("bad length of XDR frame header")
		return n, err
	}
	if (frameLen & 0x80000000) != 0x80000000 {
		err := errors.New("malformed XDR frame header")
		return n, err
	}
	frameLen &= 0x7fffffff
	m, e := xdr.Unmarshal(r, v)
	if e != nil {
		err := errors.Wrap(e, "unmarshalling framed XDR")
		return n + m, err
	}
	if int64(m) != int64(frameLen) {
		err := errors.New("bad length of XDR frame body")
		return n + m, err
	}
	return m + n, nil
}

// Returns the sequence number of an LCM, returning an error if the LCM is of
// an unknown version.
func peekLedgerSequence(xlcm *xdr.LedgerCloseMeta) (uint32, error) {
	v0, ok := xlcm.GetV0()
	if !ok {
		err := errors.New("unexpected XDR LedgerCloseMeta version")
		return 0, err
	}
	return uint32(v0.LedgerHeader.Header.LedgerSeq), nil
}

// Note: the xdr.LedgerCloseMeta structure is _not_ the same as
// the ledgerbackend.LedgerCloseMeta structure; the latter should
// probably migrate to the former eventually.
func (c *captiveStellarCore) copyLedgerCloseMeta(xlcm *xdr.LedgerCloseMeta, lcm *LedgerCloseMeta) error {
	v0, ok := xlcm.GetV0()
	if !ok {
		return errors.New("unexpected XDR LedgerCloseMeta version")
	}
	lcm.LedgerHeader = v0.LedgerHeader
	envelopes := make(map[xdr.Hash]xdr.TransactionEnvelope)
	for _, tx := range v0.TxSet.Txs {
		hash, e := network.HashTransactionInEnvelope(tx, c.networkPassphrase)
		if e != nil {
			return errors.Wrap(e, "error hashing tx in LedgerCloseMeta")
		}
		envelopes[hash] = tx
	}
	for _, trm := range v0.TxProcessing {
		txe, ok := envelopes[trm.Result.TransactionHash]
		if !ok {
			return errors.New("unknown tx hash in LedgerCloseMeta")
		}
		lcm.TransactionEnvelope = append(lcm.TransactionEnvelope, txe)
		lcm.TransactionResult = append(lcm.TransactionResult, trm.Result)
		lcm.TransactionMeta = append(lcm.TransactionMeta, trm.TxApplyProcessing)
		lcm.TransactionFeeChanges = append(lcm.TransactionFeeChanges, trm.FeeProcessing)
	}
	for _, urm := range v0.UpgradesProcessing {
		lcm.UpgradesMeta = append(lcm.UpgradesMeta, urm.Changes)
	}
	return nil
}

func (c *captiveStellarCore) openOfflineReplaySubprocess(nextLedger, lastLedger uint32) error {
	c.Close()
	maxLedger, e := c.GetLatestLedgerSequence()
	if e != nil {
		return errors.Wrap(e, "getting latest ledger sequence")
	}
	if nextLedger > maxLedger {
		err := errors.Errorf("sequence %d greater than max available %d",
			nextLedger, maxLedger)
		return err
	}
	if lastLedger > maxLedger {
		lastLedger = maxLedger
	}
	rangeArg := fmt.Sprintf("%d/%d", lastLedger, (lastLedger-nextLedger)+1)
	args := []string{"--conf", c.getConfFileName(), "catchup", rangeArg,
		"--replay-in-memory"}
	cmd := exec.Command("stellar-core", args...)
	cmd.Dir = c.getTmpDir()
	cmd.Stdout = c.getLogLineWriter()
	cmd.Stderr = cmd.Stdout
	c.cmd = cmd
	e = c.start()
	if e != nil {
		err := errors.Wrap(e, "starting stellar-core subprocess")
		return err
	}
	// The next ledger should be the first ledger of the checkpoint containing
	// the requested ledger
	c.nextLedgerMutex.Lock()
	c.nextLedger = roundDownToFirstReplayAfterCheckpointStart(nextLedger)
	c.nextLedgerMutex.Unlock()
	c.lastLedger = &lastLedger
	return nil
}

func (c *captiveStellarCore) PrepareRange(from uint32, to uint32) error {
	// `from-1` here because being able to read ledger `from-1` is a confirmation
	// that the range is ready. This effectively makes getting ledger #1 impossible.
	// TODO: should be replaced with by a tee reader with buffer or similar in the
	// later stage of development.
	if e := c.openOfflineReplaySubprocess(from-1, to); e != nil {
		return errors.Wrap(e, "opening subprocess")
	}

	if c.metaPipe == nil {
		return errors.New("missing metadata pipe")
	}

	_, _, err := c.GetLedger(from - 1)
	if err != nil {
		return errors.Wrap(err, "opening getting ledger `from-1`")
	}

	return nil
}

// We assume that we'll be called repeatedly asking for ledgers in ascending
// order, so when asked for ledger 23 we start a subprocess doing catchup
// "100023/100000", which should replay 23, 24, 25, ... 100023. The wrinkle in
// this is that core will actually replay from the _checkpoint before_
// the implicit start ledger, so we might need to skip a few ledgers until
// we hit the one requested (this routine does so transparently if needed).
func (c *captiveStellarCore) GetLedger(sequence uint32) (bool, LedgerCloseMeta, error) {
	// First, if we're open but out of range for the request, close.
	if !c.IsClosed() && !c.LedgerWithinCheckpoints(sequence, numCheckpointsLeeway) {
		c.Close()
	}

	// Next, if we're closed, open.
	if c.IsClosed() {
		if e := c.openOfflineReplaySubprocess(sequence, sequence+ledgersPerProcess); e != nil {
			return false, LedgerCloseMeta{}, errors.Wrap(e, "opening subprocess")
		}
	}

	// Check that we're where we expect to be: in range ...
	if !c.LedgerWithinCheckpoints(sequence, 1) {
		return false, LedgerCloseMeta{}, errors.New("unexpected subprocess next-ledger")
	}

	// ... and open
	if c.metaPipe == nil {
		return false, LedgerCloseMeta{}, errors.New("missing metadata pipe")
	}

	// Now loop along the range until we find the ledger we want.
	var errOut error
	for {
		var xlcm xdr.LedgerCloseMeta
		_, e0 := unmarshalFramed(c.metaPipe, &xlcm)
		if e0 != nil {
			if e0 == io.EOF {
				errOut = errors.Wrap(e0, "got EOF from subprocess")
				break
			} else {
				errOut = errors.Wrap(e0, "unmarshalling framed LedgerCloseMeta")
				break
			}
		}
		seq, e1 := peekLedgerSequence(&xlcm)
		if e1 != nil {
			errOut = e1
			break
		}
		c.nextLedgerMutex.Lock()
		if seq != c.nextLedger {
			// We got something unexpected; close and reset
			errOut = errors.Errorf("unexpected ledger %d", seq)
			c.nextLedgerMutex.Unlock()
			break
		}
		c.nextLedger++
		c.nextLedgerMutex.Unlock()
		if seq == sequence {
			// Found the requested seq
			var lcm LedgerCloseMeta
			e2 := c.copyLedgerCloseMeta(&xlcm, &lcm)
			if e2 != nil {
				errOut = e2
				break
			}
			// If we got the _last_ ledger in a segment, close before returning.
			if c.lastLedger != nil && *c.lastLedger == seq {
				c.Close()
			}
			return true, lcm, nil
		}
	}
	// All paths above that break out of the loop (instead of return)
	// set e to non-nil: there was an error and we should close and
	// reset state before retuning an error to our caller.
	c.Close()
	return false, LedgerCloseMeta{}, errOut
}

func (c *captiveStellarCore) GetLatestLedgerSequence() (uint32, error) {
	archive, e := historyarchive.Connect(
		c.historyURLs[0],
		historyarchive.ConnectOptions{},
	)
	if e != nil {
		return 0, e
	}
	has, e := archive.GetRootHAS()
	if e != nil {
		return 0, e
	}
	return has.CurrentLedger, nil
}

// LedgerWithinCheckpoints returns true if a given ledger is after the next ledger to be read
// from a given subprocess (so ledger will be read eventually) and no more
// than numCheckpoints checkpoints ahead of the next ledger to be read
// (so it will not be too long before ledger is read).
func (c *captiveStellarCore) LedgerWithinCheckpoints(ledger uint32, numCheckpoints uint32) bool {
	return ((c.nextLedger <= ledger) &&
		(ledger <= (c.nextLedger + (numCheckpoints * ledgersPerCheckpoint))))
}

func (c *captiveStellarCore) IsClosed() bool {
	c.nextLedgerMutex.Lock()
	defer c.nextLedgerMutex.Unlock()
	return c.nextLedger == 0
}

func (c *captiveStellarCore) Close() error {
	if c.IsClosed() {
		return nil
	}
	c.nextLedgerMutex.Lock()
	c.nextLedger = 0
	c.nextLedgerMutex.Unlock()

	c.lastLedger = nil
	var e1, e2 error
	if c.metaPipe != nil {
		c.metaPipe = nil
	}
	if c.processIsAlive() {
		e1 = c.cmd.Process.Kill()
		c.cmd.Wait()
		c.cmd = nil
	}
	e2 = os.RemoveAll(c.getTmpDir())
	if e1 != nil {
		return errors.Wrap(e1, "error killing subprocess")
	}
	if e2 != nil {
		return errors.Wrap(e2, "error removing subprocess tmpdir")
	}
	return nil
}

func (c *captiveStellarCore) getTmpDir() string {
	return filepath.Join(os.TempDir(), c.nonce)
}

func (c *captiveStellarCore) getConfFileName() string {
	return filepath.Join(c.getTmpDir(), "stellar-core.conf")
}

func (c *captiveStellarCore) getConf() string {
	lines := []string{
		"# Generated file -- do not edit",
		"RUN_STANDALONE=true",
		"NODE_IS_VALIDATOR=false",
		"DISABLE_XDR_FSYNC=true",
		"UNSAFE_QUORUM=true",
		fmt.Sprintf(`NETWORK_PASSPHRASE="%s"`, c.networkPassphrase),
		fmt.Sprintf(`BUCKET_DIR_PATH="%s"`, filepath.Join(c.getTmpDir(), "buckets")),
		fmt.Sprintf(`METADATA_OUTPUT_STREAM="%s"`, c.getPipeName()),
	}
	for i, val := range c.historyURLs {
		lines = append(lines, fmt.Sprintf("[HISTORY.h%d]", i))
		lines = append(lines, fmt.Sprintf(`get="curl -sf %s/{0} -o {1}"`, val))
	}
	// Add a fictional quorum -- necessary to convince core to start up;
	// but not used at all for our purposes. Pubkey here is just random.
	lines = append(lines,
		"[QUORUM_SET]",
		"THRESHOLD_PERCENT=100",
		`VALIDATORS=["GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"]`)
	return strings.ReplaceAll(strings.Join(lines, "\n"), "\\", "\\\\")
}

func (c *captiveStellarCore) getLogLineWriter() io.Writer {
	r, w := io.Pipe()
	br := bufio.NewReader(r)
	// Strip timestamps from log lines from captive stellar-core. We emit our own.
	dateRx := regexp.MustCompile("^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3} ")
	go func() {
		for {
			line, e := br.ReadString('\n')
			if e != nil {
				break
			}
			line = dateRx.ReplaceAllString(line, "")
			// Leaving for debug purposes:
			// fmt.Print(line)
		}
	}()
	return w
}

// Makes the temp directory and writes the config file to it; called by the
// platform-specific captiveStellarCore.Start() methods.
func (c *captiveStellarCore) writeConf() error {
	dir := c.getTmpDir()
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return errors.Wrap(e, "error creating subprocess tmpdir")
	}
	conf := c.getConf()
	return ioutil.WriteFile(c.getConfFileName(), []byte(conf), 0644)
}
