package ledgerbackend

import (
	"io"
	"sync"

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
	networkPassphrase string
	historyURLs       []string
	lastLedger        *uint32 // end of current segment if offline, nil if online

	stellarCoreRunner stellarCoreRunnerInterface

	nextLedgerMutex sync.Mutex
	nextLedger      uint32 // next ledger expected, error w/ restart if not seen
}

// NewCaptive returns a new captiveStellarCore that is not running. Will lazily start a subprocess
// to feed it a block of streaming metadata when user calls .GetLedger(), and will kill
// and restart the subprocess if subsequent calls to .GetLedger() are discontiguous.
//
// Platform-specific pipe setup logic is in the .start() methods.
func NewCaptive(executablePath, networkPassphrase string, historyURLs []string) *captiveStellarCore {
	return &captiveStellarCore{
		networkPassphrase: networkPassphrase,
		historyURLs:       historyURLs,
		nextLedger:        0,
		stellarCoreRunner: &stellarCoreRunner{
			executablePath:    executablePath,
			networkPassphrase: networkPassphrase,
			historyURLs:       historyURLs,
		},
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

	err := c.stellarCoreRunner.run(nextLedger, lastLedger)
	if err != nil {
		return errors.Wrap(err, "error running stellar-core")
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

	if c.stellarCoreRunner.getMetaPipe() == nil {
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
	metaPipe := c.stellarCoreRunner.getMetaPipe()
	if metaPipe == nil {
		return false, LedgerCloseMeta{}, errors.New("missing metadata pipe")
	}

	// Now loop along the range until we find the ledger we want.
	var errOut error
	for {
		var xlcm xdr.LedgerCloseMeta
		_, e0 := xdr.UnmarshalFramed(metaPipe, &xlcm)
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
			errOut = errors.Errorf("unexpected ledger (expected=%d actual=%d)", c.nextLedger, seq)
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

	err := c.stellarCoreRunner.close()
	if err != nil {
		return errors.Wrap(err, "error closing stellar-core subprocess")
	}
	return nil
}
