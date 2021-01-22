// Package ingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package ingest

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
)

const (
	// MaxSupportedProtocolVersion defines the maximum supported version of
	// the Stellar protocol.
	MaxSupportedProtocolVersion = 16

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
	// - 11: Protocol 14: CAP-23 and CAP-33.
	// - 12: Trigger state rebuild due to `absTime` -> `abs_time` rename
	//       in ClaimableBalances predicates.
	CurrentVersion = 12

	// MaxDBConnections is the size of the postgres connection pool dedicated to Horizon ingestion:
	//  * Ledger ingestion,
	//  * State verifications,
	//  * Metrics updates.
	MaxDBConnections = 3

	defaultCoreCursorName           = "HORIZON"
	stateVerificationErrorThreshold = 3
)

var log = logpkg.DefaultLogger.WithField("service", "ingest")

type Config struct {
	CoreSession                 *db.Session
	StellarCoreURL              string
	StellarCoreCursor           string
	EnableCaptiveCore           bool
	CaptiveCoreBinaryPath       string
	CaptiveCoreConfigAppendPath string
	CaptiveCoreHTTPPort         uint
	RemoteCaptiveCoreURL        string
	NetworkPassphrase           string

	HistorySession           *db.Session
	HistoryArchiveURL        string
	DisableStateVerification bool

	MaxReingestRetries          int
	ReingestRetryBackoffSeconds int

	// The checkpoint frequency will be 64 unless you are using an exotic test setup.
	CheckpointFrequency uint32
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
	// LedgerIngestionDuration exposes timing metrics about the rate and
	// duration of ledger ingestion (including updating DB and graph).
	LedgerIngestionDuration prometheus.Summary

	// StateVerifyDuration exposes timing metrics about the rate and
	// duration of state verification.
	StateVerifyDuration prometheus.Summary

	// StateInvalidGauge exposes state invalid metric. 1 if state is invalid,
	// 0 otherwise.
	StateInvalidGauge prometheus.GaugeFunc

	// LedgerStatsCounter exposes ledger stats counters (like number of ops/changes).
	LedgerStatsCounter *prometheus.CounterVec

	// ProcessorsRunDuration exposes processors run durations.
	ProcessorsRunDuration *prometheus.CounterVec
}

type System interface {
	Run()
	Metrics() Metrics
	StressTest(numTransactions, changesPerTransaction int) error
	VerifyRange(fromLedger, toLedger uint32, verifyState bool) error
	ReingestRange(fromLedger, toLedger uint32, force bool) error
	BuildGenesisState() error
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
	historyAdapter historyArchiveAdapterInterface

	stellarCoreClient stellarCoreClient

	maxReingestRetries          int
	reingestRetryBackoffSeconds int
	wg                          sync.WaitGroup

	// stateVerificationRunning is true when verification routine is currently
	// running.
	stateVerificationMutex sync.Mutex
	// number of consecutive state verification runs which encountered errors
	stateVerificationErrors  int
	stateVerificationRunning bool
	disableStateVerification bool

	checkpointManager historyarchive.CheckpointManager
}

func NewSystem(config Config) (System, error) {
	ctx, cancel := context.WithCancel(context.Background())

	archive, err := historyarchive.Connect(
		config.HistoryArchiveURL,
		historyarchive.ConnectOptions{
			Context:             ctx,
			NetworkPassphrase:   config.NetworkPassphrase,
			CheckpointFrequency: config.CheckpointFrequency,
		},
	)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "error creating history archive")
	}

	var ledgerBackend ledgerbackend.LedgerBackend
	if config.EnableCaptiveCore {
		if len(config.RemoteCaptiveCoreURL) > 0 {
			ledgerBackend, err = ledgerbackend.NewRemoteCaptive(config.RemoteCaptiveCoreURL)
			if err != nil {
				cancel()
				return nil, errors.Wrap(err, "error creating captive core backend")
			}
		} else {
			ledgerBackend, err = ledgerbackend.NewCaptive(
				ledgerbackend.CaptiveCoreConfig{
					BinaryPath:          config.CaptiveCoreBinaryPath,
					ConfigAppendPath:    config.CaptiveCoreConfigAppendPath,
					HTTPPort:            config.CaptiveCoreHTTPPort,
					NetworkPassphrase:   config.NetworkPassphrase,
					HistoryArchiveURLs:  []string{config.HistoryArchiveURL},
					CheckpointFrequency: config.CheckpointFrequency,
					LedgerHashStore:     ledgerbackend.NewHorizonDBLedgerHashStore(config.HistorySession),
					Log:                 log.WithField("subservice", "stellar-core"),
					Context:             ctx,
				},
			)
			if err != nil {
				cancel()
				return nil, errors.Wrap(err, "error creating captive core backend")
			}
		}
	} else {
		coreSession := config.CoreSession.Clone()
		coreSession.Ctx = ctx
		ledgerBackend, err = ledgerbackend.NewDatabaseBackendFromSession(coreSession, config.NetworkPassphrase)
		if err != nil {
			cancel()
			return nil, errors.Wrap(err, "error creating ledger backend")
		}
	}

	historyQ := &history.Q{config.HistorySession.Clone()}
	historyQ.Ctx = ctx

	historyAdapter := newHistoryArchiveAdapter(archive)

	system := &system{
		cancel:                      cancel,
		config:                      config,
		ctx:                         ctx,
		disableStateVerification:    config.DisableStateVerification,
		historyAdapter:              historyAdapter,
		historyQ:                    historyQ,
		ledgerBackend:               ledgerBackend,
		maxReingestRetries:          config.MaxReingestRetries,
		reingestRetryBackoffSeconds: config.ReingestRetryBackoffSeconds,
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
		checkpointManager: historyarchive.NewCheckpointManager(config.CheckpointFrequency),
	}

	system.initMetrics()
	return system, nil
}

func (s *system) initMetrics() {
	s.metrics.LedgerIngestionDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "ledger_ingestion_duration_seconds",
		Help: "ledger ingestion durations, sliding window = 10m",
	})

	s.metrics.StateVerifyDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "state_verify_duration_seconds",
		Help: "state verification durations, sliding window = 10m",
	})

	s.metrics.StateInvalidGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "state_invalid",
			Help: "equals 1 if state invalid, 0 otherwise",
		},
		func() float64 {
			invalid, err := s.historyQ.CloneIngestionQ().GetExpStateInvalid()
			if err != nil {
				log.WithError(err).Error("Error in initMetrics/GetExpStateInvalid")
				return 0
			}
			invalidFloat := float64(0)
			if invalid {
				invalidFloat = 1
			}
			return invalidFloat
		},
	)

	s.metrics.LedgerStatsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "ledger_stats_total",
			Help: "counters of different ledger stats",
		},
		[]string{"type"},
	)

	s.metrics.ProcessorsRunDuration = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "processor_run_duration_seconds_total",
			Help: "run durations of ingestion processors",
		},
		[]string{"name"},
	)
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
	run := func() error {
		return s.runStateMachine(reingestHistoryRangeState{
			fromLedger: fromLedger,
			toLedger:   toLedger,
			force:      force,
		})
	}
	err := run()
	for retry := 0; err != nil && retry < s.maxReingestRetries; retry++ {
		log.Warnf("reingest range [%d, %d] failed (%s), retrying", fromLedger, toLedger, err.Error())
		time.Sleep(time.Second * time.Duration(s.reingestRetryBackoffSeconds))
		err = run()
	}
	return err
}

// BuildGenesisState runs the ingestion pipeline on genesis ledger. Transitions
// to stopState when done.
func (s *system) BuildGenesisState() error {
	return s.runStateMachine(buildState{
		checkpointLedger: 1,
		stop:             true,
	})
}

func (s *system) runStateMachine(cur stateMachineNode) error {
	s.wg.Add(1)
	defer func() {
		s.wg.Done()
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
		s.checkpointManager.IsCheckpoint(lastIngestedLedger) { // it's a checkpoint ledger.
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
				case ingest.StateError:
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
	if s.stellarCoreClient == nil || s.config.EnableCaptiveCore {
		return nil
	}

	cursor := defaultCoreCursorName
	if s.config.StellarCoreCursor != "" {
		cursor = s.config.StellarCoreCursor
	}

	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
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
	if s.stateVerificationRunning {
		log.Info("Shutting down state verifier...")
	}
	s.stateVerificationMutex.Unlock()
	s.cancel()
	// wait for ingestion state machine to terminate
	s.wg.Wait()
	if err := s.ledgerBackend.Close(); err != nil {
		log.WithError(err).Info("could not close ledger backend")
	}
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
