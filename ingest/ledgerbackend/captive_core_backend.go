package ledgerbackend

import (
	"context"
	"encoding/hex"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

// Ensure CaptiveStellarCore implements LedgerBackend
var _ LedgerBackend = (*CaptiveStellarCore)(nil)

func (c *CaptiveStellarCore) roundDownToFirstReplayAfterCheckpointStart(ledger uint32) uint32 {
	r := c.checkpointManager.GetCheckpointRange(ledger)
	if r.Low <= 1 {
		// Stellar-Core doesn't stream ledger 1
		return 2
	}
	// All other checkpoints start at the next multiple of 64
	return r.Low
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
//     requires the configAppendPath to be provided because a quorum set needs to
//     be selected.
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
// Except for the Close function, CaptiveStellarCore is not thread-safe and should
// not be accessed by multiple go routines. Close is thread-safe and can be called
// from another go routine. Once Close is called it will interrupt and cancel any
// pending operations.
//
// Requires Stellar-Core v13.2.0+.
type CaptiveStellarCore struct {
	archive           historyarchive.ArchiveInterface
	checkpointManager historyarchive.CheckpointManager
	ledgerHashStore   TrustedLedgerHashStore

	// cancel is the CancelFunc for context which controls the lifetime of a CaptiveStellarCore instance.
	// Once it is invoked CaptiveStellarCore will not be able to stream ledgers from Stellar Core or
	// spawn new instances of Stellar Core.
	cancel context.CancelFunc

	stellarCoreRunner stellarCoreRunnerInterface
	// stellarCoreLock protects access to stellarCoreRunner. When the read lock
	// is acquired stellarCoreRunner can be accessed. When the write lock is acquired
	// stellarCoreRunner can be updated.
	stellarCoreLock sync.RWMutex

	// For testing
	stellarCoreRunnerFactory func(mode stellarCoreRunnerMode) (stellarCoreRunnerInterface, error)

	// Defines if the blocking mode (off by default) is on or off. In blocking mode,
	// calling GetLedger blocks until the requested ledger is available. This is useful
	// for scenarios when Horizon consumes ledgers faster than Stellar-Core produces them
	// and using `time.Sleep` when ledger is not available can actually slow entire
	// ingestion process.
	// blockingLock locks access to blocking.
	blockingLock sync.Mutex
	blocking     bool

	// cachedMeta keeps that ledger data of the last fetched ledger. Updated in GetLedger().
	cachedMeta *xdr.LedgerCloseMeta

	nextLedger         uint32  // next ledger expected, error w/ restart if not seen
	lastLedger         *uint32 // end of current segment if offline, nil if online
	previousLedgerHash *string
}

// CaptiveCoreConfig contains all the parameters required to create a CaptiveStellarCore instance
type CaptiveCoreConfig struct {
	// BinaryPath is the file path to the Stellar Core binary
	BinaryPath string
	// ConfigAppendPath is the file path to additional configuration for the Stellar Core configuration file used
	// by captive core. This field is only required when ingesting in online mode (e.g. UnboundedRange).
	ConfigAppendPath string
	// NetworkPassphrase is the Stellar network passphrase used by captive core when connecting to the Stellar network
	NetworkPassphrase string
	// HistoryArchiveURLs are a list of history archive urls
	HistoryArchiveURLs []string

	// Optional fields

	// CheckpointFrequency is the number of ledgers between checkpoints
	// if unset, DefaultCheckpointFrequency will be used
	CheckpointFrequency uint32
	// LedgerHashStore is an optional store used to obtain hashes for ledger sequences from a trusted source
	LedgerHashStore TrustedLedgerHashStore
	// HTTPPort is the TCP port to listen for requests (defaults to 0, which disables the HTTP server)
	HTTPPort uint
	// PeerPort is the TCP port to bind to for connecting to the Stellar network
	// (defaults to 11625). It may be useful for example when there's >1 Stellar-Core
	// instance running on a machine.
	PeerPort uint
	// Log is an (optional) custom logger which will capture any output from the Stellar Core process.
	// If Log is omitted then all output will be printed to stdout.
	Log *log.Entry
	// LogPath is the (optional) path in which to store Core logs, passed along
	// to Stellar Core's LOG_FILE_PATH
	LogPath string
	// Context is the (optional) context which controls the lifetime of a CaptiveStellarCore instance. Once the context is done
	// the CaptiveStellarCore instance will not be able to stream ledgers from Stellar Core or spawn new
	// instances of Stellar Core. If Context is omitted CaptiveStellarCore will default to using context.Background.
	Context context.Context
	// StoragePath is the (optional) base path passed along to Core's
	// BUCKET_DIR_PATH which specifies where various bucket data should be
	// stored. We always append /captive-core to this directory, since we clean
	// it up entirely on shutdown.
	StoragePath string
}

// NewCaptive returns a new CaptiveStellarCore instance.
func NewCaptive(config CaptiveCoreConfig) (*CaptiveStellarCore, error) {
	// Here we set defaults in the config. Because config is not a pointer this code should
	// not mutate the original CaptiveCoreConfig instance which was passed into NewCaptive()

	// Log Captive Core straight to stdout by default
	if config.Log == nil {
		config.Log = log.New()
		config.Log.Logger.SetOutput(os.Stdout)
		config.Log.SetLevel(logrus.InfoLevel)
	}

	parentCtx := config.Context
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	var cancel context.CancelFunc
	config.Context, cancel = context.WithCancel(parentCtx)

	archivePool, err := historyarchive.NewArchivePool(
		config.HistoryArchiveURLs,
		historyarchive.ConnectOptions{
			NetworkPassphrase:   config.NetworkPassphrase,
			CheckpointFrequency: config.CheckpointFrequency,
			Context:             config.Context,
		},
	)

	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "Error connecting to ALL history archives.")
	}

	c := &CaptiveStellarCore{
		archive:           &archivePool,
		ledgerHashStore:   config.LedgerHashStore,
		cancel:            cancel,
		checkpointManager: historyarchive.NewCheckpointManager(config.CheckpointFrequency),
	}

	c.stellarCoreRunnerFactory = func(mode stellarCoreRunnerMode) (stellarCoreRunnerInterface, error) {
		return newStellarCoreRunner(config, mode)
	}
	return c, nil
}

func (c *CaptiveStellarCore) getLatestCheckpointSequence() (uint32, error) {
	has, err := c.archive.GetRootHAS()
	if err != nil {
		return 0, errors.Wrap(err, "error getting root HAS")
	}

	return has.CurrentLedger, nil
}

func (c *CaptiveStellarCore) openOfflineReplaySubprocess(from, to uint32) error {
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

	var runner stellarCoreRunnerInterface
	if runner, err = c.stellarCoreRunnerFactory(stellarCoreRunnerModeOffline); err != nil {
		return errors.Wrap(err, "error creating stellar-core runner")
	} else {
		// only assign c.stellarCoreRunner if runner is not nil to avoid nil interface check
		// see https://golang.org/doc/faq#nil_error
		c.stellarCoreRunner = runner
	}

	err = c.stellarCoreRunner.catchup(from, to)
	if err != nil {
		return errors.Wrap(err, "error running stellar-core")
	}

	// The next ledger should be the first ledger of the checkpoint containing
	// the requested ledger
	c.nextLedger = c.roundDownToFirstReplayAfterCheckpointStart(from)
	c.lastLedger = &to
	c.setBlocking(true)
	c.previousLedgerHash = nil

	return nil
}

func (c *CaptiveStellarCore) openOnlineReplaySubprocess(from uint32) error {
	latestCheckpointSequence, err := c.getLatestCheckpointSequence()
	if err != nil {
		return errors.Wrap(err, "error getting latest checkpoint sequence")
	}

	// We don't allow starting the online mode starting with more than two
	// checkpoints from now. Such requests are likely buggy.
	// We should allow only one checkpoint here but sometimes there are up to a
	// minute delays when updating root HAS by stellar-core.
	twoCheckPointsLength := (c.checkpointManager.GetCheckpoint(0) + 1) * 2
	maxLedger := latestCheckpointSequence + twoCheckPointsLength
	if from > maxLedger {
		return errors.Errorf(
			"trying to start online mode too far (latest checkpoint=%d), only two checkpoints in the future allowed",
			latestCheckpointSequence,
		)
	}

	var runner stellarCoreRunnerInterface
	if runner, err = c.stellarCoreRunnerFactory(stellarCoreRunnerModeOnline); err != nil {
		return errors.Wrap(err, "error creating stellar-core runner")
	} else {
		// only assign c.stellarCoreRunner if runner is not nil to avoid nil interface check
		// see https://golang.org/doc/faq#nil_error
		c.stellarCoreRunner = runner
	}

	runFrom, ledgerHash, nextLedger, err := c.runFromParams(from)
	if err != nil {
		return errors.Wrap(err, "error calculating ledger and hash for stelar-core run")
	}

	err = c.stellarCoreRunner.runFrom(runFrom, ledgerHash)
	if err != nil {
		return errors.Wrap(err, "error running stellar-core")
	}

	c.nextLedger = nextLedger
	c.lastLedger = nil
	c.previousLedgerHash = nil

	if c.ledgerHashStore != nil {
		var exists bool
		ledgerHash, exists, err = c.ledgerHashStore.GetLedgerHash(nextLedger - 1)
		if err != nil {
			return errors.Wrapf(err, "error trying to read ledger hash %d", nextLedger-1)
		}
		if exists {
			c.previousLedgerHash = &ledgerHash
		}
	}

	c.setBlocking(false)

	return nil
}

// runFromParams receives a ledger sequence and calculates the required values to call stellar-core run with --start-ledger and --start-hash
func (c *CaptiveStellarCore) runFromParams(from uint32) (runFrom uint32, ledgerHash string, nextLedger uint32, err error) {
	if from == 1 {
		// Trying to start-from 1 results in an error from Stellar-Core:
		// Target ledger 1 is not newer than last closed ledger 1 - nothing to do
		// TODO maybe we can fix it by generating 1st ledger meta
		// like GenesisLedgerStateReader?
		err = errors.New("CaptiveCore is unable to start from ledger 1, start from ledger 2")
		return
	}

	if from <= 63 {
		// For ledgers before (and including) first checkpoint, get/wait the first
		// checkpoint to get the ledger header. It will always start streaming
		// from ledger 2.
		nextLedger = 2
		// The line below is to support a special case for streaming ledger 2
		// that works for all other ledgers <= 63 (fast-forward).
		// We can't set from=2 because Stellar-Core will not allow starting from 1.
		// To solve this we start from 3 and exploit the fast that Stellar-Core
		// will stream data from 2 for the first checkpoint.
		from = 3
	} else {
		// For ledgers after the first checkpoint, start at the previous checkpoint
		// and fast-forward from there.
		if !c.checkpointManager.IsCheckpoint(from) {
			from = c.checkpointManager.PrevCheckpoint(from)
		}
		// Streaming will start from the previous checkpoint + 1
		nextLedger = from - 63
		if nextLedger < 2 {
			// Stellar-Core always streams from ledger 2 at min.
			nextLedger = 2
		}
	}

	runFrom = from - 1
	if c.ledgerHashStore != nil {
		var exists bool
		ledgerHash, exists, err = c.ledgerHashStore.GetLedgerHash(runFrom)
		if err != nil {
			err = errors.Wrapf(err, "error trying to read ledger hash %d", runFrom)
			return
		}
		if exists {
			return
		}
	}

	ledgerHeader, err2 := c.archive.GetLedgerHeader(from)
	if err2 != nil {
		err = errors.Wrapf(err2, "error trying to read ledger header %d from HAS", from)
		return
	}
	ledgerHash = hex.EncodeToString(ledgerHeader.Header.PreviousLedgerHash[:])
	return
}

func (c *CaptiveStellarCore) startPreparingRange(ledgerRange Range) (bool, error) {
	c.stellarCoreLock.Lock()
	defer c.stellarCoreLock.Unlock()

	if c.isPrepared(ledgerRange) {
		return true, nil
	}

	if c.stellarCoreRunner != nil {
		if err := c.stellarCoreRunner.close(); err != nil {
			return false, errors.Wrap(err, "error closing existing session")
		}
	}

	var err error
	if ledgerRange.bounded {
		err = c.openOfflineReplaySubprocess(ledgerRange.from, ledgerRange.to)
	} else {
		err = c.openOnlineReplaySubprocess(ledgerRange.from)
	}
	if err != nil {
		return false, errors.Wrap(err, "opening subprocess")
	}

	return false, nil
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
	if alreadyPrepared, err := c.startPreparingRange(ledgerRange); err != nil {
		return errors.Wrap(err, "error starting prepare range")
	} else if alreadyPrepared {
		return nil
	}

	old := c.isBlocking()
	c.setBlocking(true)
	_, _, err := c.GetLedger(ledgerRange.from)
	c.setBlocking(old)

	if err != nil {
		return errors.Wrapf(err, "Error fast-forwarding to %d", ledgerRange.from)
	}

	return nil
}

// IsPrepared returns true if a given ledgerRange is prepared.
func (c *CaptiveStellarCore) IsPrepared(ledgerRange Range) (bool, error) {
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

	return c.isPrepared(ledgerRange), nil
}

func (c *CaptiveStellarCore) isPrepared(ledgerRange Range) bool {
	if c.isClosed() {
		return false
	}

	lastLedger := uint32(0)
	if c.lastLedger != nil {
		lastLedger = *c.lastLedger
	}

	cachedLedger := uint32(0)
	if c.cachedMeta != nil {
		cachedLedger = c.cachedMeta.LedgerSequence()
	}

	if c.nextLedger == 0 {
		return false
	}

	if lastLedger == 0 {
		return c.nextLedger <= ledgerRange.from || cachedLedger == ledgerRange.from
	}

	// From now on: lastLedger != 0 so current range is bounded

	if ledgerRange.bounded {
		return (c.nextLedger <= ledgerRange.from || cachedLedger == ledgerRange.from) &&
			lastLedger >= ledgerRange.to
	}

	// Requested range is unbounded but current one is bounded
	return false
}

// GetLedgerBlocking works as GetLedger but will block until the ledger is
// available in the backend (even for UnboundedRange).
// Please note that requesting a ledger sequence far after current ledger will
// block the execution for a long time.
func (c *CaptiveStellarCore) GetLedgerBlocking(sequence uint32) (xdr.LedgerCloseMeta, error) {
	old := c.isBlocking()
	c.setBlocking(true)
	_, meta, err := c.GetLedger(sequence)
	c.setBlocking(old)
	return meta, err
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
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

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

	if c.lastLedger != nil && sequence > *c.lastLedger {
		return false, xdr.LedgerCloseMeta{}, errors.Errorf(
			"reading past bounded range (requested sequence=%d, last ledger in range=%d)",
			sequence,
			*c.lastLedger,
		)
	}

	// Now loop along the range until we find the ledger we want.
	var errOut error
	for {
		if !c.isBlocking() && len(c.stellarCoreRunner.getMetaPipe()) == 0 {
			return false, xdr.LedgerCloseMeta{}, nil
		}

		result, ok := <-c.stellarCoreRunner.getMetaPipe()
		if errOut = c.checkMetaPipeResult(result, ok); errOut != nil {
			break
		}

		seq := result.LedgerCloseMeta.LedgerSequence()
		if seq != c.nextLedger {
			// We got something unexpected; close and reset
			errOut = errors.Errorf(
				"unexpected ledger sequence (expected=%d actual=%d)",
				c.nextLedger,
				seq,
			)
			break
		}

		newPreviousLedgerHash := result.LedgerCloseMeta.PreviousLedgerHash().HexString()
		if c.previousLedgerHash != nil && *c.previousLedgerHash != newPreviousLedgerHash {
			// We got something unexpected; close and reset
			errOut = errors.Errorf(
				"unexpected previous ledger hash for ledger %d (expected=%s actual=%s)",
				seq,
				*c.previousLedgerHash,
				newPreviousLedgerHash,
			)
			break
		}

		c.nextLedger++
		currentLedgerHash := result.LedgerCloseMeta.LedgerHash().HexString()
		c.previousLedgerHash = &currentLedgerHash

		// Update cache with the latest value because we incremented nextLedger.
		c.cachedMeta = result.LedgerCloseMeta

		if seq == sequence {
			// If we got the _last_ ledger in a segment, close before returning.
			if c.lastLedger != nil && *c.lastLedger == seq {
				if err := c.stellarCoreRunner.close(); err != nil {
					return false, xdr.LedgerCloseMeta{}, errors.Wrap(err, "error closing session")
				}
			}
			return true, *c.cachedMeta, nil
		}
	}
	// All paths above that break out of the loop (instead of return)
	// set errOut to non-nil: there was an error and we should close and
	// reset state before retuning an error to our caller.
	c.stellarCoreRunner.close()
	return false, xdr.LedgerCloseMeta{}, errOut
}

func (c *CaptiveStellarCore) checkMetaPipeResult(result metaResult, ok bool) error {
	// There are 3 types of errors we check for:
	// 1. User initiated shutdown by canceling the parent context or calling Close().
	// 2. The stellar core process exited unexpectedly.
	// 3. Some error was encountered while consuming the ledgers emitted by captive core (e.g. parsing invalid xdr)
	if err := c.stellarCoreRunner.context().Err(); err != nil {
		// Case 1 - User initiated shutdown by canceling the parent context or calling Close()
		return err
	}
	if !ok || result.err != nil {
		if exited, err := c.stellarCoreRunner.getProcessExitError(); exited {
			// Case 2 - The stellar core process exited unexpectedly
			if err == nil {
				return errors.Errorf("stellar core exited unexpectedly")
			} else {
				return errors.Wrap(err, "stellar core exited unexpectedly")
			}
		} else if !ok {
			// This case should never happen because the ledger buffer channel can only be closed
			// if and only if the process exits or the context is cancelled.
			// However, we add this check for the sake of completeness
			return errors.Errorf("meta pipe closed unexpectedly")
		} else {
			// Case 3 - Some error was encountered while consuming the ledger stream emitted by captive core.
			return result.err
		}
	}
	return nil
}

// GetLatestLedgerSequence returns the sequence of the latest ledger available
// in the backend. This method returns an error if not in a session (start with
// PrepareRange).
//
// Note that for UnboundedRange the returned sequence number is not necessarily
// the latest sequence closed by the network. It's always the last value available
// in the backend.
func (c *CaptiveStellarCore) GetLatestLedgerSequence() (uint32, error) {
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

	if c.isClosed() {
		return 0, errors.New("stellar-core must be opened to return latest available sequence")
	}

	if c.lastLedger == nil {
		return c.nextLedger - 1 + uint32(len(c.stellarCoreRunner.getMetaPipe())), nil
	}
	return *c.lastLedger, nil
}

func (c *CaptiveStellarCore) isClosed() bool {
	return c.nextLedger == 0 || c.stellarCoreRunner == nil || c.stellarCoreRunner.context().Err() != nil
}

func (c *CaptiveStellarCore) isBlocking() bool {
	c.blockingLock.Lock()
	defer c.blockingLock.Unlock()
	return c.blocking
}

func (c *CaptiveStellarCore) setBlocking(val bool) {
	c.blockingLock.Lock()
	c.blocking = val
	c.blockingLock.Unlock()
}

// Close closes existing Stellar-Core process, streaming sessions and removes all
// temporary files. Note, once a CaptiveStellarCore instance is closed it can can no longer be used and
// all subsequent calls to PrepareRange(), GetLedger(), etc will fail.
// Close is thread-safe and can be called from another go routine.
func (c *CaptiveStellarCore) Close() error {
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

	// after the CaptiveStellarCore context is canceled all subsequent calls to PrepareRange() will fail
	c.cancel()

	if c.stellarCoreRunner != nil {
		return c.stellarCoreRunner.close()
	}
	return nil
}
