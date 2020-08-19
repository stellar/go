package ledgerbackend

import (
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

// Ensure CaptiveStellarCore implements LedgerBackend
var _ LedgerBackend = (*CaptiveStellarCore)(nil)

const (
	readAheadBufferSize = 2
)

func roundDownToFirstReplayAfterCheckpointStart(ledger uint32) uint32 {
	v := (ledger / ledgersPerCheckpoint) * ledgersPerCheckpoint
	if v == 0 {
		// Stellar-Core doesn't stream ledger 1
		return 2
	}
	// All other checkpoints start at the next multiple of 64
	return v
}

type metaResult struct {
	*xdr.LedgerCloseMeta
	err error
}

// CaptiveStellarCore is a ledger backend that starts internal Stellar-Core
// subprocess responsible for streaming ledger data. It provides better decoupling
// than DatabaseBackend but requires some extra init time.
//
// It operates in two modes:
//   * When a BoundedRange is prepared it starts Stellar-Core in catchup mode that
//     replays ledgers in memory. This is very fast but requires Stellar-Core to
//     keep ledger state in RAM. It requires around 3GB of RAM as of August 2020.
//   * When a UnboundedRange is prepared it runs Stellar-Core catchup mode to
//     sync with the first ledger and then runs it in a normal mode. This
//     requires the configPath to be provided because a database connection is
//     required and quorum set needs to be selected.
//
// The database requirement for UnboundedRange will soon be removed when some
// changes are implemented in Stellar-Core.
//
// When running CaptiveStellarCore will create a temporary folder to store
// bucket files and other temporary files. The folder is removed when Close is
// called.
//
// The communication is performed via filesystem pipe which is created in a
// temporary folder.
//
// Currently BoundedRange requires a full-trust on history archive. This issue is
// being fixed in Stellar-Core.
//
// While using BoundedRanges is straightforward there are a few gotchas connected
// to UnboundedRanges:
//   * PrepareRange takes more time because all ledger entries must be stored on
//     disk instead of RAM.
//   * If GetLedger is not called frequently (every 5 sec. on average) the
//     Stellar-Core process can go out of sync with the network. This happens
//     because there is no buffering of communication pipe and CaptiveStellarCore
//     has a very small internal buffer and Stellar-Core will not close the new
//     ledger if it's not read.
//
// Requires Stellar-Core v13.2.0+.
type CaptiveStellarCore struct {
	executablePath    string
	configPath        string
	networkPassphrase string
	historyURLs       []string
	archive           historyarchive.ArchiveInterface

	// shutdown is a channel that triggers the backend shutdown by the user.
	shutdown chan struct{}
	// metaC is a read-ahead buffer.
	metaC chan metaResult
	// wait is a waiting group waiting for a read-ahead buffer to return.
	wait sync.WaitGroup

	stellarCoreRunner stellarCoreRunnerInterface
	cachedMeta        *xdr.LedgerCloseMeta

	// Defines if the blocking mode (off by default) is on or off. In blocking mode,
	// calling GetLedger blocks until the requested ledger is available. This is useful
	// for scenarios when Horizon consumes ledgers faster than Stellar-Core produces them
	// and using `time.Sleep` when ledger is not available can actually slow entire
	// ingestion process.
	blocking bool

	nextLedger uint32  // next ledger expected, error w/ restart if not seen
	lastLedger *uint32 // end of current segment if offline, nil if online

	processExitMutex sync.Mutex
	processExit      bool
	processErr       error

	// waitIntervalPrepareRange defines a time to wait between checking if the buffer
	// is empty. Default 1s, lower in tests to make them faster.
	waitIntervalPrepareRange time.Duration
}

// NewCaptive returns a new CaptiveStellarCore.
//
// All parameters are required, except configPath which is not required when
// working with BoundedRanges only.
func NewCaptive(executablePath, configPath, networkPassphrase string, historyURLs []string) (*CaptiveStellarCore, error) {
	archive, err := historyarchive.Connect(
		historyURLs[0],
		historyarchive.ConnectOptions{},
	)
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to history archive")
	}

	return &CaptiveStellarCore{
		archive:                  archive,
		executablePath:           executablePath,
		configPath:               configPath,
		historyURLs:              historyURLs,
		networkPassphrase:        networkPassphrase,
		waitIntervalPrepareRange: time.Second,
	}, nil
}

func (c *CaptiveStellarCore) getLatestCheckpointSequence() (uint32, error) {
	has, err := c.archive.GetRootHAS()
	if err != nil {
		return 0, errors.Wrap(err, "error getting root HAS")
	}

	return has.CurrentLedger, nil
}

func (c *CaptiveStellarCore) openOfflineReplaySubprocess(from, to uint32) error {
	err := c.Close()
	if err != nil {
		return errors.Wrap(err, "error closing existing session")
	}

	latestCheckpointSequence, err := c.getLatestCheckpointSequence()
	if err != nil {
		return errors.Wrap(err, "error getting latest checkpoint sequence")
	}
	if from > latestCheckpointSequence {
		return errors.Errorf(
			"sequence: %d is greater than max available in history archives: %d",
			from,
			latestCheckpointSequence,
		)
	}
	if to > latestCheckpointSequence {
		to = latestCheckpointSequence
	}

	if c.stellarCoreRunner == nil {
		// configPath is empty in an offline mode because it's generated
		c.stellarCoreRunner, err = newStellarCoreRunner(c.executablePath, "", c.networkPassphrase, c.historyURLs)
		if err != nil {
			return errors.Wrap(err, "error creating stellar-core runner")
		}
	}
	err = c.stellarCoreRunner.catchup(from, to)
	if err != nil {
		return errors.Wrap(err, "error running stellar-core")
	}

	// The next ledger should be the first ledger of the checkpoint containing
	// the requested ledger
	c.nextLedger = roundDownToFirstReplayAfterCheckpointStart(from)
	c.lastLedger = &to
	c.blocking = true
	c.processExit = false
	c.processErr = nil

	// read-ahead buffer
	c.metaC = make(chan metaResult, readAheadBufferSize)
	c.shutdown = make(chan struct{})
	c.wait.Add(1)
	go c.sendLedgerMeta(to)
	return nil
}

func (c *CaptiveStellarCore) openOnlineReplaySubprocess(from uint32) error {
	// Check if existing session works for this request
	if c.lastLedger == nil && c.nextLedger != 0 && c.nextLedger <= from {
		// Use existing session, GetLedger will fast-forward
		return nil
	}

	err := c.Close()
	if err != nil {
		return errors.Wrap(err, "error closing existing session")
	}

	latestCheckpointSequence, err := c.getLatestCheckpointSequence()
	if err != nil {
		return errors.Wrap(err, "error getting latest checkpoint sequence")
	}

	// We don't allow starting the online mode starting with more than two
	// checkpoints from now. Such requests are likely buggy.
	// We should allow only one checkpoint here but sometimes there are up to a
	// minute delays when updating root HAS by stellar-core.
	maxLedger := latestCheckpointSequence + 2*64
	if from > maxLedger {
		return errors.Errorf(
			"trying to start online mode too far (latest checkpoint=%d), only two checkpoints in the future allowed",
			latestCheckpointSequence,
		)
	}

	if c.stellarCoreRunner == nil {
		if c.configPath == "" {
			return errors.New("stellar-core config file path cannot be empty in an online mode")
		}
		c.stellarCoreRunner, err = newStellarCoreRunner(c.executablePath, c.configPath, c.networkPassphrase, c.historyURLs)
		if err != nil {
			return errors.Wrap(err, "error creating stellar-core runner")
		}
	}
	err = c.stellarCoreRunner.runFrom(from)
	if err != nil {
		return errors.Wrap(err, "error running stellar-core")
	}

	// The next ledger should be the ledger actually requested because
	// we run `catchup X/0` command in the online mode.
	c.nextLedger = from
	c.lastLedger = nil
	c.blocking = false
	c.processExit = false
	c.processErr = nil

	// read-ahead buffer
	c.metaC = make(chan metaResult, readAheadBufferSize)
	c.shutdown = make(chan struct{})
	c.wait.Add(1)
	go c.sendLedgerMeta(0)
	return nil
}

// sendLedgerMeta reads from the captive core pipe, decodes the ledger metadata
// and sends it to the metadata buffered channel
func (c *CaptiveStellarCore) sendLedgerMeta(untilSequence uint32) {
	defer c.wait.Done()
	printBufferOccupation := time.NewTicker(5 * time.Second)
	defer printBufferOccupation.Stop()
	for {
		select {
		case <-c.shutdown:
			return
		case <-printBufferOccupation.C:
			log.Debug("captive core read-ahead buffer occupation:", len(c.metaC))
		default:
		}

		meta, err := c.readLedgerMetaFromPipe()
		if err != nil {
			select {
			case processErr := <-c.stellarCoreRunner.getProcessExitChan():
				// First, check if this is an error caused by a process exit.
				c.processExitMutex.Lock()
				c.processExit = true
				c.processErr = processErr
				c.processExitMutex.Unlock()
				if processErr != nil {
					err = errors.Wrap(processErr, "stellar-core process exited with an error")
				} else {
					err = errors.New("stellar-core process exited without an error unexpectedly")
				}
			default:
			}
			// When `GetLedger` sees the error it will close the backend. We shouldn't
			// close it now because there may be some ledgers in a buffer.
			c.metaC <- metaResult{nil, err}
			return
		}
		c.metaC <- metaResult{meta, nil}

		if untilSequence != 0 {
			if meta.LedgerSequence() >= untilSequence {
				// we are done
				return
			}
		}
	}
}

func (c *CaptiveStellarCore) readLedgerMetaFromPipe() (*xdr.LedgerCloseMeta, error) {
	metaPipe := c.stellarCoreRunner.getMetaPipe()
	if metaPipe == nil {
		return nil, errors.New("missing metadata pipe")
	}
	var xlcm xdr.LedgerCloseMeta
	_, e0 := xdr.UnmarshalFramed(metaPipe, &xlcm)
	if e0 != nil {
		if e0 == io.EOF {
			return nil, errors.Wrap(e0, "got EOF from subprocess")
		} else {
			return nil, errors.Wrap(e0, "unmarshalling framed LedgerCloseMeta")
		}
	}
	return &xlcm, nil
}

// PrepareRange prepares the given range (including from and to) to be loaded.
// Captive stellar-core backend needs to initalize Stellar-Core state to be
// able to stream ledgers.
// Stellar-Core mode depends on the provided ledgerRange:
//   * For BoundedRange it will start Stellar-Core in catchup mode.
//   * For UnboundedRange it will first catchup to starting ledger and then run
//     it normally (including connecting to the Stellar network).
// Please note that using a BoundedRange, currently, requires a full-trust on
// history archive. This issue is being fixed in Stellar-Core.
func (c *CaptiveStellarCore) PrepareRange(ledgerRange Range) error {
	// Range already prepared
	if prepared, err := c.IsPrepared(ledgerRange); err != nil {
		return errors.Wrap(err, "error in IsPrepared")
	} else if prepared {
		return nil
	}

	var err error
	if ledgerRange.bounded {
		err = c.openOfflineReplaySubprocess(ledgerRange.from, ledgerRange.to)
	} else {
		err = c.openOnlineReplaySubprocess(ledgerRange.from)
	}
	if err != nil {
		return errors.Wrap(err, "opening subprocess")
	}

	metaPipe := c.stellarCoreRunner.getMetaPipe()
	if metaPipe == nil {
		return errors.New("missing metadata pipe")
	}

	for {
		select {
		case <-c.shutdown:
			return nil
		default:
		}
		// Wait for the first ledger or an error
		if len(c.metaC) > 0 {
			// If process exited return an error
			c.processExitMutex.Lock()
			if c.processExit {
				if c.processErr != nil {
					err = errors.Wrap(c.processErr, "stellar-core process exited with an error")
				} else {
					err = errors.New("stellar-core process exited without an error unexpectedly")
				}
			}
			c.processExitMutex.Unlock()
			if err != nil {
				return err
			}
			break
		}
		time.Sleep(c.waitIntervalPrepareRange)
	}

	return nil
}

// IsPrepared returns true if a given ledgerRange is prepared.
func (c *CaptiveStellarCore) IsPrepared(ledgerRange Range) (bool, error) {
	if c.nextLedger == 0 {
		return false, nil
	}

	if c.lastLedger == nil {
		return c.nextLedger <= ledgerRange.from, nil
	}

	// From now on: c.lastLedger != nil so current range is bounded

	if ledgerRange.bounded {
		return c.nextLedger <= ledgerRange.from &&
			c.nextLedger <= *c.lastLedger, nil
	}

	// Requested range is unbounded but current one is bounded
	return false, nil
}

// GetLedger returns true when ledger is found and it's LedgerCloseMeta.
// Call PrepareRange first to instruct the backend which ledgers to fetch.
//
// CaptiveStellarCore requires PrepareRange call first to initialize Stellar-Core.
// Requesting a ledger on non-prepared backend will return an error.
//
// Because data is streamed from Stellar-Core ledger after ledger user should
// request sequences in a non-decreasing order. If the requested sequence number
// is less than the last requested sequence number, an error will be returned.
//
// This function behaves differently for bounded and unbounded ranges:
//   * BoundedRange makes GetLedger blocking if the requested ledger is not yet
//     available in the ledger. After getting the last ledger in a range this
//     method will also Close() the backend.
//   * UnboundedRange makes GetLedger non-blocking. The method will return with
//     the first argument equal false.
// This is done to provide maximum performance when streaming old ledgers.
func (c *CaptiveStellarCore) GetLedger(sequence uint32) (bool, xdr.LedgerCloseMeta, error) {
	if c.cachedMeta != nil && sequence == c.cachedMeta.LedgerSequence() {
		// GetLedger can be called multiple times using the same sequence, ex. to create
		// change and transaction readers. If we have this ledger buffered, let's return it.
		return true, *c.cachedMeta, nil
	}

	if c.isClosed() {
		return false, xdr.LedgerCloseMeta{}, errors.New("session is closed, call PrepareRange first")
	}

	if sequence < c.nextLedger {
		return false, xdr.LedgerCloseMeta{}, errors.Errorf(
			"requested ledger %d is behind the captive core stream (expected=%d)",
			sequence,
			c.nextLedger,
		)
	}

	// Now loop along the range until we find the ledger we want.
	var errOut error
loop:
	for {
		if !c.blocking && len(c.metaC) == 0 {
			return false, xdr.LedgerCloseMeta{}, nil
		}

		metaResult := <-c.metaC
		if metaResult.err != nil {
			errOut = metaResult.err
			break loop
		}

		seq := metaResult.LedgerCloseMeta.LedgerSequence()
		if seq != c.nextLedger {
			// We got something unexpected; close and reset
			errOut = errors.Errorf("unexpected ledger (expected=%d actual=%d)", c.nextLedger, seq)
			break
		}
		c.nextLedger++
		if seq == sequence {
			// Found the requested seq
			c.cachedMeta = metaResult.LedgerCloseMeta

			// If we got the _last_ ledger in a segment, close before returning.
			if c.lastLedger != nil && *c.lastLedger == seq {
				if err := c.Close(); err != nil {
					return false, xdr.LedgerCloseMeta{}, errors.Wrap(err, "error closing session")
				}
			}
			return true, *c.cachedMeta, nil
		}
	}
	// All paths above that break out of the loop (instead of return)
	// set e to non-nil: there was an error and we should close and
	// reset state before retuning an error to our caller.
	c.Close()
	return false, xdr.LedgerCloseMeta{}, errOut
}

// GetLatestLedgerSequence returns the sequence of the latest ledger available
// in the backend. This method returns an error if not in a session (start with
// PrepareRange).
//
// Note that for UnboundedRange the returned sequence number is not necessarily
// the latest sequence closed by the network. It's always the last value available
// in the backend.
func (c *CaptiveStellarCore) GetLatestLedgerSequence() (uint32, error) {
	if c.isClosed() {
		return 0, errors.New("stellar-core must be opened to return latest available sequence")
	}

	if c.lastLedger == nil {
		return c.nextLedger - 1 + uint32(len(c.metaC)), nil
	}
	return *c.lastLedger, nil
}

func (c *CaptiveStellarCore) isClosed() bool {
	return c.nextLedger == 0
}

// Close closes existing Stellar-Core process, streaming sessions and removes all
// temporary files.
func (c *CaptiveStellarCore) Close() error {
	if c.isClosed() {
		return nil
	}
	c.nextLedger = 0
	c.lastLedger = nil

	if c.shutdown != nil {
		close(c.shutdown)
		// discard pending data in case the goroutine is blocked writing to the channel,
		// see: `sendLedgerMeta`.
		select {
		case <-c.metaC:
		default:
		}
		// Do not close the communication channel until we know
		// the goroutine is done
		c.wait.Wait()
		close(c.metaC)
	}

	if c.stellarCoreRunner != nil {
		err := c.stellarCoreRunner.close()
		if err != nil {
			return errors.Wrap(err, "error closing stellar-core subprocess")
		}
	}
	return nil
}
