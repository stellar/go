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

	// cachedMeta keeps that ledger data of the last fetched ledger. Updated in GetLedger().
	cachedMeta *xdr.LedgerCloseMeta

	prepared           *Range  // non-nil if any range is prepared
	closed             bool    // False until the core is closed
	nextLedger         uint32  // next ledger expected, error w/ restart if not seen
	lastLedger         *uint32 // end of current segment if offline, nil if online
	previousLedgerHash *string
}

// CaptiveCoreConfig contains all the parameters required to create a CaptiveStellarCore instance
type CaptiveCoreConfig struct {
	// BinaryPath is the file path to the Stellar Core binary
	BinaryPath string
	// NetworkPassphrase is the Stellar network passphrase used by captive core when connecting to the Stellar network
	NetworkPassphrase string
	// HistoryArchiveURLs are a list of history archive urls
	HistoryArchiveURLs []string
	Toml               *CaptiveCoreToml

	// Optional fields

	// CheckpointFrequency is the number of ledgers between checkpoints
	// if unset, DefaultCheckpointFrequency will be used
	CheckpointFrequency uint32
	// LedgerHashStore is an optional store used to obtain hashes for ledger sequences from a trusted source
	LedgerHashStore TrustedLedgerHashStore
	// Log is an (optional) custom logger which will capture any output from the Stellar Core process.
	// If Log is omitted then all output will be printed to stdout.
	Log *log.Entry
	// Context is the (optional) context which controls the lifetime of a CaptiveStellarCore instance. Once the context is done
	// the CaptiveStellarCore instance will not be able to stream ledgers from Stellar Core or spawn new
	// instances of Stellar Core. If Context is omitted CaptiveStellarCore will default to using context.Background.
	Context context.Context
	// StoragePath is the (optional) base path passed along to Core's
	// BUCKET_DIR_PATH which specifies where various bucket data should be
	// stored. We always append /captive-core to this directory, since we clean
	// it up entirely on shutdown.
	StoragePath string

	// UseDB, when true, instructs the core invocation to use an external db url
	// for ledger states rather than in memory(RAM). The external db url is determined by the presence
	// of DATABASE parameter in the captive-core-config-path or if absent, the db will default to sqlite
	// and the db file will be stored at location derived from StoragePath parameter.
	UseDB bool
}

// NewCaptive returns a new CaptiveStellarCore instance.
func NewCaptive(config CaptiveCoreConfig) (*CaptiveStellarCore, error) {
	// Here we set defaults in the config. Because config is not a pointer this code should
	// not mutate the original CaptiveCoreConfig instance which was passed into NewCaptive()

	// Log Captive Core straight to stdout by default
	if config.Log == nil {
		config.Log = log.New()
		config.Log.SetOutput(os.Stdout)
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
			"from sequence: %d is greater than max available in history archives: %d",
			from,
			latestCheckpointSequence,
		)
	}

	if to > latestCheckpointSequence {
		return errors.Errorf(
			"to sequence: %d is greater than max available in history archives: %d",
			to,
			latestCheckpointSequence,
		)
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
	ran := BoundedRange(from, to)
	c.prepared = &ran
	c.nextLedger = c.roundDownToFirstReplayAfterCheckpointStart(from)
	c.lastLedger = &to
	c.previousLedgerHash = nil

	return nil
}

func (c *CaptiveStellarCore) openOnlineReplaySubprocess(ctx context.Context, from uint32) error {
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
	}
	c.stellarCoreRunner = runner

	runFrom, ledgerHash, err := c.runFromParams(ctx, from)
	if err != nil {
		return errors.Wrap(err, "error calculating ledger and hash for stellar-core run")
	}

	err = c.stellarCoreRunner.runFrom(runFrom, ledgerHash)
	if err != nil {
		return errors.Wrap(err, "error running stellar-core")
	}

	// In the online mode we update nextLedger after streaming the first ledger.
	// This is to support versions before and after/including v17.1.0 that
	// introduced minimal persistent DB.
	c.nextLedger = 0
	ran := UnboundedRange(from)
	c.prepared = &ran
	c.lastLedger = nil
	c.previousLedgerHash = nil

	return nil
}

// runFromParams receives a ledger sequence and calculates the required values to call stellar-core run with --start-ledger and --start-hash
func (c *CaptiveStellarCore) runFromParams(ctx context.Context, from uint32) (uint32, string, error) {

	if from == 1 {
		// Trying to start-from 1 results in an error from Stellar-Core:
		// Target ledger 1 is not newer than last closed ledger 1 - nothing to do
		// TODO maybe we can fix it by generating 1st ledger meta
		// like GenesisLedgerStateReader?
		err := errors.New("CaptiveCore is unable to start from ledger 1, start from ledger 2")
		return 0, "", err
	}

	if from <= 63 {
		// The line below is to support a special case for streaming ledger 2
		// that works for all other ledgers <= 63 (fast-forward).
		// We can't set from=2 because Stellar-Core will not allow starting from 1.
		// To solve this we start from 3 and exploit the fast that Stellar-Core
		// will stream data from 2 for the first checkpoint.
		from = 3
	}

	runFrom := from - 1
	if c.ledgerHashStore != nil {
		var exists bool
		ledgerHash, exists, err := c.ledgerHashStore.GetLedgerHash(ctx, runFrom)
		if err != nil {
			err = errors.Wrapf(err, "error trying to read ledger hash %d", runFrom)
			return 0, "", err
		}
		if exists {
			return runFrom, ledgerHash, nil
		}
	}

	ledgerHeader, err := c.archive.GetLedgerHeader(from)
	if err != nil {
		return 0, "", errors.Wrapf(err, "error trying to read ledger header %d from HAS", from)
	}
	ledgerHash := hex.EncodeToString(ledgerHeader.Header.PreviousLedgerHash[:])
	return runFrom, ledgerHash, nil
}

// nextExpectedSequence returns nextLedger (if currently set) or start of
// prepared range. Otherwise it returns 0.
// This is done because `nextLedger` is 0 between the moment Stellar-Core is
// started and streaming the first ledger (in such case we return first ledger
// in requested range).
func (c *CaptiveStellarCore) nextExpectedSequence() uint32 {
	if c.nextLedger == 0 && c.prepared != nil {
		return c.prepared.from
	}
	return c.nextLedger
}

func (c *CaptiveStellarCore) startPreparingRange(ctx context.Context, ledgerRange Range) (bool, error) {
	c.stellarCoreLock.Lock()
	defer c.stellarCoreLock.Unlock()

	if c.isPrepared(ledgerRange) {
		return true, nil
	}

	if c.stellarCoreRunner != nil {
		if err := c.stellarCoreRunner.close(); err != nil {
			return false, errors.Wrap(err, "error closing existing session")
		}

		// Make sure Stellar-Core is terminated before starting a new instance.
		processExited, _ := c.stellarCoreRunner.getProcessExitError()
		if !processExited {
			return false, errors.New("the previous Stellar-Core instance is still running")
		}
	}

	var err error
	if ledgerRange.bounded {
		err = c.openOfflineReplaySubprocess(ledgerRange.from, ledgerRange.to)
	} else {
		err = c.openOnlineReplaySubprocess(ctx, ledgerRange.from)
	}
	if err != nil {
		return false, errors.Wrap(err, "opening subprocess")
	}

	return false, nil
}

// PrepareRange prepares the given range (including from and to) to be loaded.
// Captive stellar-core backend needs to initialize Stellar-Core state to be
// able to stream ledgers.
// Stellar-Core mode depends on the provided ledgerRange:
//   * For BoundedRange it will start Stellar-Core in catchup mode.
//   * For UnboundedRange it will first catchup to starting ledger and then run
//     it normally (including connecting to the Stellar network).
// Please note that using a BoundedRange, currently, requires a full-trust on
// history archive. This issue is being fixed in Stellar-Core.
func (c *CaptiveStellarCore) PrepareRange(ctx context.Context, ledgerRange Range) error {
	if alreadyPrepared, err := c.startPreparingRange(ctx, ledgerRange); err != nil {
		return errors.Wrap(err, "error starting prepare range")
	} else if alreadyPrepared {
		return nil
	}

	_, err := c.GetLedger(ctx, ledgerRange.from)
	if err != nil {
		return errors.Wrapf(err, "Error fast-forwarding to %d", ledgerRange.from)
	}

	return nil
}

// IsPrepared returns true if a given ledgerRange is prepared.
func (c *CaptiveStellarCore) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

	return c.isPrepared(ledgerRange), nil
}

func (c *CaptiveStellarCore) isPrepared(ledgerRange Range) bool {
	if c.closed {
		return false
	}

	if c.stellarCoreRunner == nil || c.stellarCoreRunner.context().Err() != nil {
		return false
	}

	if exited, _ := c.stellarCoreRunner.getProcessExitError(); exited {
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

	if c.prepared == nil {
		return false
	}

	if lastLedger == 0 {
		return c.nextExpectedSequence() <= ledgerRange.from || cachedLedger == ledgerRange.from
	}

	// From now on: lastLedger != 0 so current range is bounded

	if ledgerRange.bounded {
		return (c.nextExpectedSequence() <= ledgerRange.from || cachedLedger == ledgerRange.from) &&
			lastLedger >= ledgerRange.to
	}

	// Requested range is unbounded but current one is bounded
	return false
}

// GetLedger will block until the ledger is available in the backend
// (even for UnboundedRange), then return it's LedgerCloseMeta.
//
// Call PrepareRange first to instruct the backend which ledgers to fetch.
// CaptiveStellarCore requires PrepareRange call first to initialize Stellar-Core.
// Requesting a ledger on non-prepared backend will return an error.
//
// Please note that requesting a ledger sequence far after current
// ledger will block the execution for a long time.
//
// Because ledger data is streamed from Stellar-Core sequentially, users should
// request sequences in a non-decreasing order. If the requested sequence number
// is less than the last requested sequence number, an error will be returned.
//
// This function behaves differently for bounded and unbounded ranges:
//   * BoundedRange: After getting the last ledger in a range this method will
//     also Close() the backend.
func (c *CaptiveStellarCore) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

	if c.cachedMeta != nil && sequence == c.cachedMeta.LedgerSequence() {
		// GetLedger can be called multiple times using the same sequence, ex. to create
		// change and transaction readers. If we have this ledger buffered, let's return it.
		return *c.cachedMeta, nil
	}

	if c.closed {
		return xdr.LedgerCloseMeta{}, errors.New("stellar-core is no longer usable")
	}

	if c.prepared == nil {
		return xdr.LedgerCloseMeta{}, errors.New("session is not prepared, call PrepareRange first")
	}

	if c.stellarCoreRunner == nil {
		return xdr.LedgerCloseMeta{}, errors.New("stellar-core cannot be nil, call PrepareRange first")
	}
	if c.closed {
		return xdr.LedgerCloseMeta{}, errors.New("stellar-core has an error, call PrepareRange first")
	}

	if sequence < c.nextExpectedSequence() {
		return xdr.LedgerCloseMeta{}, errors.Errorf(
			"requested ledger %d is behind the captive core stream (expected=%d)",
			sequence,
			c.nextExpectedSequence(),
		)
	}

	if c.lastLedger != nil && sequence > *c.lastLedger {
		return xdr.LedgerCloseMeta{}, errors.Errorf(
			"reading past bounded range (requested sequence=%d, last ledger in range=%d)",
			sequence,
			*c.lastLedger,
		)
	}

	// Now loop along the range until we find the ledger we want.
	for {
		select {
		case <-ctx.Done():
			return xdr.LedgerCloseMeta{}, ctx.Err()
		case result, ok := <-c.stellarCoreRunner.getMetaPipe():
			found, ledger, err := c.handleMetaPipeResult(sequence, result, ok)
			if found || err != nil {
				return ledger, err
			}
		}
	}
}

func (c *CaptiveStellarCore) handleMetaPipeResult(sequence uint32, result metaResult, ok bool) (bool, xdr.LedgerCloseMeta, error) {
	if err := c.checkMetaPipeResult(result, ok); err != nil {
		c.stellarCoreRunner.close()
		return false, xdr.LedgerCloseMeta{}, err
	}

	seq := result.LedgerCloseMeta.LedgerSequence()
	// If we got something unexpected; close and reset
	if c.nextLedger != 0 && seq != c.nextLedger {
		c.stellarCoreRunner.close()
		return false, xdr.LedgerCloseMeta{}, errors.Errorf(
			"unexpected ledger sequence (expected=%d actual=%d)",
			c.nextLedger,
			seq,
		)
	} else if c.nextLedger == 0 && seq > c.prepared.from {
		// First stream ledger is greater than prepared.from
		c.stellarCoreRunner.close()
		return false, xdr.LedgerCloseMeta{}, errors.Errorf(
			"unexpected ledger sequence (expected=<=%d actual=%d)",
			c.prepared.from,
			seq,
		)
	}

	newPreviousLedgerHash := result.LedgerCloseMeta.PreviousLedgerHash().HexString()
	if c.previousLedgerHash != nil && *c.previousLedgerHash != newPreviousLedgerHash {
		// We got something unexpected; close and reset
		c.stellarCoreRunner.close()
		return false, xdr.LedgerCloseMeta{}, errors.Errorf(
			"unexpected previous ledger hash for ledger %d (expected=%s actual=%s)",
			seq,
			*c.previousLedgerHash,
			newPreviousLedgerHash,
		)
	}

	c.nextLedger = result.LedgerSequence() + 1
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

	return false, xdr.LedgerCloseMeta{}, nil
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
		if result.err != nil {
			// Case 3 - Some error was encountered while consuming the ledger stream emitted by captive core.
			return result.err
		} else if exited, err := c.stellarCoreRunner.getProcessExitError(); exited {
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
func (c *CaptiveStellarCore) GetLatestLedgerSequence(ctx context.Context) (uint32, error) {
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

	if c.closed {
		return 0, errors.New("stellar-core is no longer usable")
	}
	if c.prepared == nil {
		return 0, errors.New("stellar-core must be prepared, call PrepareRange first")
	}
	if c.stellarCoreRunner == nil {
		return 0, errors.New("stellar-core cannot be nil, call PrepareRange first")
	}
	if c.closed {
		return 0, errors.New("stellar-core is closed, call PrepareRange first")

	}
	if c.lastLedger == nil {
		return c.nextExpectedSequence() - 1 + uint32(len(c.stellarCoreRunner.getMetaPipe())), nil
	}
	return *c.lastLedger, nil
}

// Close closes existing Stellar-Core process, streaming sessions and removes all
// temporary files. Note, once a CaptiveStellarCore instance is closed it can no longer be used and
// all subsequent calls to PrepareRange(), GetLedger(), etc will fail.
// Close is thread-safe and can be called from another go routine.
func (c *CaptiveStellarCore) Close() error {
	c.stellarCoreLock.RLock()
	defer c.stellarCoreLock.RUnlock()

	c.closed = true

	// after the CaptiveStellarCore context is canceled all subsequent calls to PrepareRange() will fail
	c.cancel()

	// TODO: Sucks to ignore the error here, but no worse than it was before,
	// so...
	if c.ledgerHashStore != nil {
		c.ledgerHashStore.Close()
	}

	if c.stellarCoreRunner != nil {
		return c.stellarCoreRunner.close()
	}
	return nil
}
