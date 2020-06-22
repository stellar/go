// Package expingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package expingest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/exp/ingest/adapters"
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	logpkg "github.com/stellar/go/support/log"
)

const (
	// CurrentVersion reflects the latest version of the ingestion
	// algorithm. This value is stored in KV store and is used to decide
	// if there's a need to reprocess the ledger state or reingest data.
	//
	// Version history:
	// - 1: Initial version
	// - 2: Added the orderbook, offers processors and distributed ingestion.
	// - 3: Fixed a bug that could potentialy result in invalid state
	//      (#1722). Update the version to clear the state.
	// - 4: Fixed a bug in AccountSignersChanged method.
	// - 5: Added trust lines.
	// - 6: Added accounts and accounts data.
	// - 7: Fixes a bug in AccountSignersChanged method.
	// - 8: Fixes AccountSigners processor to remove preauth tx signer
	//      when preauth tx is failed.
	// - 9: Fixes a bug in asset stats processor that counted unauthorized
	//      trustlines.
	// - 10: Fixes a bug in meta processing (fees are now processed before
	//      everything else).
	CurrentVersion = 10

	// MaxDBConnections is the size of the postgres connection pool dedicated to Horizon ingestion
	MaxDBConnections = 2

	defaultCoreCursorName           = "HORIZON"
	stateVerificationErrorThreshold = 3
)

var log = logpkg.DefaultLogger.WithField("service", "expingest")

type Config struct {
	CoreSession       *db.Session
	StellarCoreURL    string
	StellarCoreCursor string
	StellarCorePath   string
	NetworkPassphrase string

	HistorySession           *db.Session
	HistoryArchiveURL        string
	DisableStateVerification bool

	// MaxStreamRetries determines how many times the reader will retry when encountering
	// errors while streaming xdr bucket entries from the history archive.
	// Set MaxStreamRetries to 0 if there should be no retry attempts
	MaxStreamRetries int
}

const (
	getLastIngestedErrMsg           string = "Error getting last ingested ledger"
	getExpIngestVersionErrMsg       string = "Error getting exp ingest version"
	updateLastLedgerExpIngestErrMsg string = "Error updating last ingested ledger"
	commitErrMsg                    string = "Error committing db transaction"
	updateExpStateInvalidErrMsg     string = "Error updating state invalid value"
)

type stellarCoreClient interface {
	SetCursor(ctx context.Context, id string, cursor int32) error
}

type Metrics struct {
	// LedgerIngestionTimer exposes timing metrics about the rate and
	// duration of ledger ingestion (including updating DB and graph).
	LedgerIngestionTimer metrics.Timer

	// LedgerInMemoryIngestionTimer exposes timing metrics about the rate and
	// duration of ingestion into in-memory graph only.
	LedgerInMemoryIngestionTimer metrics.Timer

	// StateVerifyTimer exposes timing metrics about the rate and
	// duration of state verification.
	StateVerifyTimer metrics.Timer
}

type System interface {
	Run()
	Metrics() Metrics
	StressTest(numTransactions, changesPerTransaction int) error
	VerifyRange(fromLedger, toLedger uint32, verifyState bool) error
	ReingestRange(fromLedger, toLedger uint32, force bool) error
	Shutdown()
}

type system struct {
	metrics Metrics
	ctx     context.Context
	cancel  context.CancelFunc

	config Config

	historyQ history.IngestionQ
	runner   ProcessorRunnerInterface

	ledgerBackend  ledgerbackend.LedgerBackend
	historyAdapter adapters.HistoryArchiveAdapterInterface

	stellarCoreClient stellarCoreClient

	maxStreamRetries int
	wg               sync.WaitGroup

	// stateVerificationRunning is true when verification routine is currently
	// running.
	stateVerificationMutex sync.Mutex
	// number of consecutive state verification runs which encountered errors
	stateVerificationErrors  int
	stateVerificationRunning bool
	disableStateVerification bool
}

func NewSystem(config Config) (System, error) {
	ctx, cancel := context.WithCancel(context.Background())

	archive, err := historyarchive.Connect(
		config.HistoryArchiveURL,
		historyarchive.ConnectOptions{
			Context: ctx,
		},
	)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "error creating history archive")
	}

	coreSession := config.CoreSession.Clone()
	coreSession.Ctx = ctx

	var ledgerBackend ledgerbackend.LedgerBackend
	if len(config.StellarCorePath) > 0 {
		ledgerBackend = ledgerbackend.NewCaptive(
			config.StellarCorePath,
			config.NetworkPassphrase,
			[]string{config.HistoryArchiveURL},
		)
	} else {
		ledgerBackend, err = ledgerbackend.NewDatabaseBackendFromSession(coreSession, config.NetworkPassphrase)
		if err != nil {
			cancel()
			return nil, errors.Wrap(err, "error creating ledger backend")
		}
	}

	historyQ := &history.Q{config.HistorySession.Clone()}
	historyQ.Ctx = ctx

	historyAdapter := adapters.MakeHistoryArchiveAdapter(archive)

	system := &system{
		ctx:                      ctx,
		cancel:                   cancel,
		historyAdapter:           historyAdapter,
		ledgerBackend:            ledgerBackend,
		config:                   config,
		historyQ:                 historyQ,
		disableStateVerification: config.DisableStateVerification,
		maxStreamRetries:         config.MaxStreamRetries,
		stellarCoreClient: &stellarcore.Client{
			URL: config.StellarCoreURL,
		},
		runner: &ProcessorRunner{
			ctx:            ctx,
			config:         config,
			historyQ:       historyQ,
			historyAdapter: historyAdapter,
			ledgerBackend:  ledgerBackend,
		},
	}

	system.initMetrics()
	return system, nil
}

func (s *system) initMetrics() {
	s.metrics.LedgerIngestionTimer = metrics.NewTimer()
	s.metrics.LedgerInMemoryIngestionTimer = metrics.NewTimer()
	s.metrics.StateVerifyTimer = metrics.NewTimer()
}

func (s *system) Metrics() Metrics {
	return s.metrics
}

// Run starts ingestion system. Ingestion system supports distributed ingestion
// that means that Horizon ingestion can be running on multiple machines and
// only one, random node will lead the ingestion.
//
// It needs to support cartesian product of the following run scenarios cases:
// - Init from empty state (1a) and resuming from existing state (1b).
// - Ingestion system version has been upgraded (2a) or not (2b).
// - Current node is leading ingestion (3a) or not (3b).
//
// We always clear state when ingestion system is upgraded so 2a and 2b are
// included in 1a.
//
// We ensure that only one instance is a leader because in each round instances
// try to acquire a lock on `LastLedgerExpIngest value in key value store and only
// one instance will be able to acquire it. This happens in both initial processing
// and ledger processing. So this solves 3a and 3b in both 1a and 1b.
//
// Finally, 1a and 1b are tricky because we need to keep the latest version
// of order book graph in memory of each Horizon instance. To solve this:
// * For state init:
//   * If instance is a leader, we update the order book graph by running state
//     pipeline normally.
//   * If instance is NOT a leader, we build a graph from offers present in a
//     database. We completely omit state pipeline in this case.
// * For resuming:
//   * If instances is a leader, it runs full ledger pipeline, including updating
//     a database.
//   * If instances is a NOT leader, it runs ledger pipeline without updating a
//     a database so order book graph is updated but database is not overwritten.
func (s *system) Run() {
	s.runStateMachine(startState{})
}

func (s *system) StressTest(numTransactions, changesPerTransaction int) error {
	if numTransactions <= 0 {
		return errors.New("transactions must be positive")
	}
	if changesPerTransaction <= 0 {
		return errors.New("changes per transaction must be positive")
	}

	s.runner.EnableMemoryStatsLogging()
	s.runner.SetLedgerBackend(fakeLedgerBackend{
		numTransactions:       numTransactions,
		changesPerTransaction: changesPerTransaction,
	})
	return s.runStateMachine(stressTestState{})
}

// VerifyRange runs the ingestion pipeline on the range of ledgers. When
// verifyState is true it verifies the state when ingestion is complete.
func (s *system) VerifyRange(fromLedger, toLedger uint32, verifyState bool) error {
	return s.runStateMachine(verifyRangeState{
		fromLedger:  fromLedger,
		toLedger:    toLedger,
		verifyState: verifyState,
	})
}

// ReingestRange runs the ingestion pipeline on the range of ledgers ingesting
// history data only.
func (s *system) ReingestRange(fromLedger, toLedger uint32, force bool) error {
	return s.runStateMachine(reingestHistoryRangeState{
		fromLedger: fromLedger,
		toLedger:   toLedger,
		force:      force,
	})
}

func (s *system) runStateMachine(cur stateMachineNode) error {
	defer func() {
		s.wg.Wait()
	}()

	log.WithFields(logpkg.F{"current_state": cur}).Info("Ingestion system initial state")

	for {
		// Every node in the state machine is responsible for
		// creating and disposing its own transaction.
		// We should never enter a new state with the transaction
		// from the previous state.
		if s.historyQ.GetTx() != nil {
			panic("unexpected transaction")
		}

		next, err := cur.run(s)
		if err != nil {
			logger := log.WithFields(logpkg.F{
				"error":         err,
				"current_state": cur,
				"next_state":    next.node,
			})
			if isCancelledError(err) {
				// We only expect context.Canceled errors to occur when horizon is shutting down
				// so we log these errors using the info log level
				logger.Info("Error in ingestion state machine")
			} else {
				logger.Error("Error in ingestion state machine")
			}
		}

		// Exit after processing shutdownState
		if next.node == (stopState{}) {
			log.Info("Shut down")
			return err
		}

		select {
		case <-s.ctx.Done():
			log.Info("Received shut down signal...")
			return nil
		case <-time.After(next.sleepDuration):
		}

		log.WithFields(logpkg.F{
			"current_state": cur,
			"next_state":    next.node,
		}).Info("Ingestion system state machine transition")
		cur = next.node
	}
}

func (s *system) maybeVerifyState(lastIngestedLedger uint32) {
	stateInvalid, err := s.historyQ.GetExpStateInvalid()
	if err != nil && !isCancelledError(err) {
		log.WithField("err", err).Error("Error getting state invalid value")
	}

	// Run verification routine only when...
	if !stateInvalid && // state has not been proved to be invalid...
		!s.disableStateVerification && // state verification is not disabled...
		historyarchive.IsCheckpoint(lastIngestedLedger) { // it's a checkpoint ledger.
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			err := s.verifyState(true)
			if err != nil {
				if isCancelledError(err) {
					return
				}

				errorCount := s.incrementStateVerificationErrors()
				switch errors.Cause(err).(type) {
				case ingesterrors.StateError:
					markStateInvalid(s.historyQ, err)
				default:
					logger := log.WithField("err", err).Warn
					if errorCount >= stateVerificationErrorThreshold {
						logger = log.WithField("err", err).Error
					}
					logger("State verification errored")
				}
			} else {
				s.resetStateVerificationErrors()
			}
		}()
	}
}

func (s *system) incrementStateVerificationErrors() int {
	s.stateVerificationMutex.Lock()
	defer s.stateVerificationMutex.Unlock()
	s.stateVerificationErrors++
	return s.stateVerificationErrors
}

func (s *system) resetStateVerificationErrors() {
	s.stateVerificationMutex.Lock()
	defer s.stateVerificationMutex.Unlock()
	s.stateVerificationErrors = 0
}

func (s *system) updateCursor(ledgerSequence uint32) error {
	if s.stellarCoreClient == nil {
		return nil
	}

	cursor := defaultCoreCursorName
	if s.config.StellarCoreCursor != "" {
		cursor = s.config.StellarCoreCursor
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := s.stellarCoreClient.SetCursor(ctx, cursor, int32(ledgerSequence))
	if err != nil {
		return errors.Wrap(err, "Setting stellar-core cursor failed")
	}

	return nil
}

func (s *system) Shutdown() {
	log.Info("Shutting down ingestion system...")
	s.stateVerificationMutex.Lock()
	defer s.stateVerificationMutex.Unlock()
	if s.stateVerificationRunning {
		log.Info("Shutting down state verifier...")
	}
	s.cancel()
}

func markStateInvalid(historyQ history.IngestionQ, err error) {
	log.WithField("err", err).Error("STATE IS INVALID!")
	q := historyQ.CloneIngestionQ()
	if err := q.UpdateExpStateInvalid(true); err != nil {
		log.WithField("err", err).Error(updateExpStateInvalidErrMsg)
	}
}

func isCancelledError(err error) bool {
	cause := errors.Cause(err)
	return cause == context.Canceled || cause == db.ErrCancelled
}

type ledgerRange struct {
	from uint32
	to   uint32
}

type rangeResult struct {
	err            error
	requestedRange ledgerRange
}

type ParallelSystems struct {
	workerCount       uint
	reingestJobQueue  chan ledgerRange
	shutdown          chan struct{}
	wait              sync.WaitGroup
	reingestJobResult chan rangeResult
}

func NewParallelSystems(config Config, workerCount uint) (*ParallelSystems, error) {
	return newParallelSystems(config, workerCount, NewSystem)
}

// private version of NewParallel systems, allowing to inject a mock system
func newParallelSystems(config Config, workerCount uint, systemFactory func(Config) (System, error)) (*ParallelSystems, error) {
	if workerCount < 1 {
		return nil, errors.New("workerCount must be > 0")
	}

	result := ParallelSystems{
		workerCount:       workerCount,
		reingestJobQueue:  make(chan ledgerRange),
		shutdown:          make(chan struct{}),
		reingestJobResult: make(chan rangeResult),
	}
	for i := uint(0); i < workerCount; i++ {
		s, err := systemFactory(config)
		if err != nil {
			result.Shutdown()
			return nil, errors.Wrap(err, "cannot create new system")
		}
		result.wait.Add(1)
		go result.work(s)
	}
	return &result, nil
}

func (ps *ParallelSystems) work(s System) {
	defer func() {
		s.Shutdown()
		ps.wait.Done()
	}()
	for {
		select {
		case <-ps.shutdown:
			return
		case ledgerRange := <-ps.reingestJobQueue:
			err := s.ReingestRange(ledgerRange.from, ledgerRange.to, false)
			select {
			case <-ps.shutdown:
				return
			case ps.reingestJobResult <- rangeResult{err, ledgerRange}:
			}
		}
	}
}

const (
	historyCheckpointLedgerInterval = 64
	minBatchSize                    = historyCheckpointLedgerInterval
)

func calculateParallelLedgerBatchSize(rangeSize uint32, batchSizeSuggestion uint32, workerCount uint) uint32 {
	batchSize := batchSizeSuggestion
	if batchSize == 0 || rangeSize/batchSize < uint32(workerCount) {
		// let's try to make use of all the workers
		batchSize = rangeSize / uint32(workerCount)
	}
	// Use a minimum batch size to make it worth it in terms of overhead
	if batchSize < minBatchSize {
		batchSize = minBatchSize
	}

	// Also, round the batch size to the closest, lower or equal 64 multiple
	return (batchSize / historyCheckpointLedgerInterval) * historyCheckpointLedgerInterval
}

func (ps *ParallelSystems) ReingestRange(fromLedger, toLedger uint32, batchSizeSuggestion uint32) error {
	batchSize := calculateParallelLedgerBatchSize(toLedger-fromLedger, batchSizeSuggestion, ps.workerCount)
	pendingJobsCount := 0
	var result error
	processSubRangeResult := func(subRangeResult rangeResult) {
		pendingJobsCount--
		if result == nil && subRangeResult.err != nil {
			// TODO: give account of what ledgers were correctly reingested?
			errMsg := fmt.Sprintf("in subrange %d to %d",
				subRangeResult.requestedRange.from, subRangeResult.requestedRange.to)
			result = errors.Wrap(subRangeResult.err, errMsg)
		}
	}

	for subRangeFrom := fromLedger; subRangeFrom < toLedger && result == nil; {
		// job queuing
		subRangeTo := subRangeFrom + batchSize
		if subRangeTo > toLedger {
			subRangeTo = toLedger
		}
		subRange := ledgerRange{subRangeFrom, subRangeTo}

		select {
		case <-ps.shutdown:
			return errors.New("aborted")
		case subRangeResult := <-ps.reingestJobResult:
			processSubRangeResult(subRangeResult)
		case ps.reingestJobQueue <- subRange:
			pendingJobsCount++
			subRangeFrom = subRangeTo
		}
	}

	for pendingJobsCount > 0 {
		// wait for any remaining running jobs to finish
		select {
		case <-ps.shutdown:
			return errors.New("aborted")
		case subRangeResult := <-ps.reingestJobResult:
			processSubRangeResult(subRangeResult)
		}

	}

	return result
}

func (ps *ParallelSystems) Shutdown() {
	if ps.shutdown != nil {
		close(ps.shutdown)
		ps.wait.Wait()
		close(ps.reingestJobQueue)
		close(ps.reingestJobResult)
	}
}
