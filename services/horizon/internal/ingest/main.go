// Package ingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package ingest

import (
	"context"
	"fmt"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/filters"
	apkg "github.com/stellar/go/support/app"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"
	"github.com/stellar/go/xdr"
)

const (
	// MaxSupportedProtocolVersion defines the maximum supported version of
	// the Stellar protocol.
	MaxSupportedProtocolVersion uint32 = 21

	// CurrentVersion reflects the latest version of the ingestion
	// algorithm. This value is stored in KV store and is used to decide
	// if there's a need to reprocess the ledger state or reingest data.
	//
	// Version history:
	// - 1: Initial version
	// - 2: Added the orderbook, offers processors and distributed ingestion.
	// - 3: Fixed a bug that could potentially result in invalid state
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
	// - 13: Trigger state rebuild to include more than just authorized assets.
	// - 14: Trigger state rebuild to include claimable balances in the asset stats processor.
	// - 15: Fixed bug in asset stat ingestion where clawback is enabled (#3846).
	// - 16: Extract claimants to a separate table for better performance of
	//       claimable balances for claimant queries.
	// - 17: Add contract_id column to exp_asset_stats table which is derived by ingesting
	//       contract data ledger entries.
	// - 18: Ingest contract asset balances so we can keep track of expired / restore asset
	//       balances for asset stats.
	CurrentVersion = 18

	// MaxDBConnections is the size of the postgres connection pool dedicated to Horizon ingestion:
	//  * Ledger ingestion,
	//  * State verifications,
	//  * Metrics updates.
	MaxDBConnections = 3

	defaultCoreCursorName           = "HORIZON"
	stateVerificationErrorThreshold = 3

	// 100 ledgers per flush has shown in stress tests
	// to be best point on performance curve, default to that.
	MaxLedgersPerFlush uint32 = 100
)

var log = logpkg.DefaultLogger.WithField("service", "ingest")

type Config struct {
	StellarCoreURL         string
	CaptiveCoreBinaryPath  string
	CaptiveCoreStoragePath string
	CaptiveCoreToml        *ledgerbackend.CaptiveCoreToml
	CaptiveCoreConfigUseDB bool
	NetworkPassphrase      string

	HistorySession        db.SessionInterface
	HistoryArchiveURLs    []string
	HistoryArchiveCaching bool

	DisableStateVerification     bool
	EnableReapLookupTables       bool
	EnableExtendedLogLedgerStats bool

	MaxReingestRetries          int
	ReingestRetryBackoffSeconds int

	// The checkpoint frequency will be 64 unless you are using an exotic test setup.
	CheckpointFrequency                  uint32
	StateVerificationCheckpointFrequency uint32
	StateVerificationTimeout             time.Duration

	RoundingSlippageFilter int

	MaxLedgerPerFlush uint32
	SkipTxmeta        bool
}

const (
	getLastIngestedErrMsg        string = "Error getting last ingested ledger"
	getIngestVersionErrMsg       string = "Error getting ingestion version"
	updateLastLedgerIngestErrMsg string = "Error updating last ingested ledger"
	commitErrMsg                 string = "Error committing db transaction"
	updateExpStateInvalidErrMsg  string = "Error updating state invalid value"
)

type stellarCoreClient interface {
	SetCursor(ctx context.Context, id string, cursor int32) error
}

type Metrics struct {
	// MaxSupportedProtocolVersion exposes the maximum protocol version
	// supported by this version.
	MaxSupportedProtocolVersion prometheus.Gauge

	// LocalLedger exposes the last ingested ledger by this ingesting instance.
	LocalLatestLedger prometheus.Gauge

	// LedgerIngestionDuration exposes timing metrics about the rate and
	// duration of ledger ingestion (including updating DB and graph).
	LedgerIngestionDuration prometheus.Summary

	// LedgerIngestionTradeAggregationDuration exposes timing metrics about the rate and
	// duration of rebuilding trade aggregation buckets.
	LedgerIngestionTradeAggregationDuration prometheus.Summary

	// LedgerIngestionReapLookupTablesDuration exposes timing metrics about the rate and
	// duration of reaping lookup tables.
	LedgerIngestionReapLookupTablesDuration prometheus.Summary

	// StateVerifyDuration exposes timing metrics about the rate and
	// duration of state verification.
	StateVerifyDuration prometheus.Summary

	// StateInvalidGauge exposes state invalid metric. 1 if state is invalid,
	// 0 otherwise.
	StateInvalidGauge prometheus.GaugeFunc

	// StateVerifyLedgerEntriesCount exposes total number of ledger entries
	// checked by the state verifier by type.
	StateVerifyLedgerEntriesCount *prometheus.GaugeVec

	// LedgerStatsCounter exposes ledger stats counters (like number of ops/changes).
	LedgerStatsCounter *prometheus.CounterVec

	// ProcessorsRunDuration exposes processors run durations.
	// Deprecated in favor of: ProcessorsRunDurationSummary.
	ProcessorsRunDuration *prometheus.CounterVec

	// ProcessorsRunDurationSummary exposes processors run durations.
	ProcessorsRunDurationSummary *prometheus.SummaryVec

	// LoadersRunDurationSummary exposes run durations for the ingestion loaders.
	LoadersRunDurationSummary *prometheus.SummaryVec

	// LoadersRunDurationSummary exposes stats for the ingestion loaders.
	LoadersStatsSummary *prometheus.SummaryVec

	// ArchiveRequestCounter counts how many http requests are sent to history server
	HistoryArchiveStatsCounter *prometheus.CounterVec
}

type System interface {
	Run()
	RegisterMetrics(*prometheus.Registry)
	Metrics() Metrics
	StressTest(numTransactions, changesPerTransaction int) error
	VerifyRange(fromLedger, toLedger uint32, verifyState bool) error
	BuildState(sequence uint32, skipChecks bool) error
	ReingestRange(ledgerRanges []history.LedgerRange, force bool, rebuildTradeAgg bool) error
	BuildGenesisState() error
	Shutdown()
	GetCurrentState() State
	RebuildTradeAggregationBuckets(fromLedger, toLedger uint32) error
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

	runStateVerificationOnLedger func(uint32) bool

	reapOffsets       map[string]int64
	maxLedgerPerFlush uint32

	currentStateMutex sync.Mutex
	currentState      State
}

func NewSystem(config Config) (System, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cachingPath := ""
	if config.HistoryArchiveCaching {
		cachingPath = path.Join(config.CaptiveCoreStoragePath, "bucket-cache")
	}

	archive, err := historyarchive.NewArchivePool(
		config.HistoryArchiveURLs,
		historyarchive.ArchiveOptions{
			NetworkPassphrase:   config.NetworkPassphrase,
			CheckpointFrequency: config.CheckpointFrequency,
			ConnectOptions: storage.ConnectOptions{
				Context:   ctx,
				UserAgent: fmt.Sprintf("horizon/%s golang/%s", apkg.Version(), runtime.Version()),
			},
			CachePath: cachingPath,
		},
	)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "error creating history archive")
	}

	// the only ingest option is local captive core config
	logger := log.WithField("subservice", "stellar-core")
	ledgerBackend, err := ledgerbackend.NewCaptive(
		ledgerbackend.CaptiveCoreConfig{
			BinaryPath:          config.CaptiveCoreBinaryPath,
			StoragePath:         config.CaptiveCoreStoragePath,
			UseDB:               config.CaptiveCoreConfigUseDB,
			Toml:                config.CaptiveCoreToml,
			NetworkPassphrase:   config.NetworkPassphrase,
			HistoryArchiveURLs:  config.HistoryArchiveURLs,
			CheckpointFrequency: config.CheckpointFrequency,
			LedgerHashStore:     ledgerbackend.NewHorizonDBLedgerHashStore(config.HistorySession),
			Log:                 logger,
			Context:             ctx,
			UserAgent:           fmt.Sprintf("captivecore horizon/%s golang/%s", apkg.Version(), runtime.Version()),
		},
	)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "error creating captive core backend")
	}

	historyQ := &history.Q{config.HistorySession.Clone()}
	historyAdapter := newHistoryArchiveAdapter(archive)
	filters := filters.NewFilters()

	maxLedgersPerFlush := config.MaxLedgerPerFlush
	if maxLedgersPerFlush < 1 {
		maxLedgersPerFlush = MaxLedgersPerFlush
	}

	system := &system{
		cancel:                      cancel,
		config:                      config,
		ctx:                         ctx,
		currentState:                None,
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
			session:        historyQ,
			historyAdapter: historyAdapter,
			filters:        filters,
		},
		runStateVerificationOnLedger: ledgerEligibleForStateVerification(
			config.CheckpointFrequency,
			config.StateVerificationCheckpointFrequency,
		),
		maxLedgerPerFlush: maxLedgersPerFlush,
	}

	system.initMetrics()
	return system, nil
}

func ledgerEligibleForStateVerification(checkpointFrequency, stateVerificationFrequency uint32) func(ledger uint32) bool {
	stateVerificationCheckpointManager := historyarchive.NewCheckpointManager(
		checkpointFrequency * stateVerificationFrequency,
	)
	return func(ledger uint32) bool {
		return stateVerificationCheckpointManager.IsCheckpoint(ledger)
	}
}

func (s *system) initMetrics() {
	s.metrics.MaxSupportedProtocolVersion = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "max_supported_protocol_version",
		Help: "the maximum protocol version supported by this version.",
	})

	s.metrics.MaxSupportedProtocolVersion.Set(float64(MaxSupportedProtocolVersion))

	s.metrics.LocalLatestLedger = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "local_latest_ledger",
		Help: "sequence number of the latest ledger ingested by this ingesting instance",
	})

	s.metrics.LedgerIngestionDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "ledger_ingestion_duration_seconds",
		Help:       "ledger ingestion durations, sliding window = 10m",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})

	s.metrics.LedgerIngestionTradeAggregationDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "ledger_ingestion_trade_aggregation_duration_seconds",
		Help:       "ledger ingestion trade aggregation rebuild durations, sliding window = 10m",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})

	s.metrics.LedgerIngestionReapLookupTablesDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "ledger_ingestion_reap_lookup_tables_duration_seconds",
		Help:       "ledger ingestion reap lookup tables durations, sliding window = 10m",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})

	s.metrics.StateVerifyDuration = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "horizon", Subsystem: "ingest", Name: "state_verify_duration_seconds",
		Help:       "state verification durations, sliding window = 10m",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})

	s.metrics.StateInvalidGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "state_invalid",
			Help: "equals 1 if state invalid, 0 otherwise",
		},
		func() float64 {
			invalid, err := s.historyQ.CloneIngestionQ().GetExpStateInvalid(s.ctx)
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

	s.metrics.StateVerifyLedgerEntriesCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "state_verify_ledger_entries",
			Help: "number of ledger entries downloaded from buckets in a single state verifier run",
		},
		[]string{"type"},
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

	s.metrics.ProcessorsRunDurationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "processor_run_duration_seconds",
			Help:       "run durations of ingestion processors, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"name"},
	)

	s.metrics.LoadersRunDurationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "loader_run_duration_seconds",
			Help:       "run durations of ingestion loaders, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"name"},
	)

	s.metrics.LoadersStatsSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "loader_stats",
			Help:       "stats from ingestion loaders, sliding window = 10m",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"name", "stat"},
	)

	s.metrics.HistoryArchiveStatsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "ingest", Name: "history_archive_stats_total",
			Help: "Counters of different history archive requests.  " +
				"'source' label will provide name/address of the physical history archive server from the pool for which a request may be sent.  " +
				"'type' label will further categorize the potential request into specific requests, " +
				"'file_downloads' - the count of files downloaded from an archive server, " +
				"'file_uploads' - the count of files uploaded to an archive server, " +
				"'requests' - the count of all http requests(includes both queries and file downloads) sent to an archive server, " +
				"'cache_hits' - the count of requests for an archive file that were found on local cache instead, no download request sent to archive server.",
		},
		[]string{"source", "type"},
	)
}

func (s *system) GetCurrentState() State {
	s.currentStateMutex.Lock()
	defer s.currentStateMutex.Unlock()
	return s.currentState
}

func (s *system) Metrics() Metrics {
	return s.metrics
}

// RegisterMetrics registers the prometheus metrics
func (s *system) RegisterMetrics(registry *prometheus.Registry) {
	registry.MustRegister(s.metrics.MaxSupportedProtocolVersion)
	registry.MustRegister(s.metrics.LocalLatestLedger)
	registry.MustRegister(s.metrics.LedgerIngestionDuration)
	registry.MustRegister(s.metrics.LedgerIngestionTradeAggregationDuration)
	registry.MustRegister(s.metrics.LedgerIngestionReapLookupTablesDuration)
	registry.MustRegister(s.metrics.StateVerifyDuration)
	registry.MustRegister(s.metrics.StateInvalidGauge)
	registry.MustRegister(s.metrics.LedgerStatsCounter)
	registry.MustRegister(s.metrics.ProcessorsRunDuration)
	registry.MustRegister(s.metrics.ProcessorsRunDurationSummary)
	registry.MustRegister(s.metrics.LoadersRunDurationSummary)
	registry.MustRegister(s.metrics.LoadersStatsSummary)
	registry.MustRegister(s.metrics.StateVerifyLedgerEntriesCount)
	registry.MustRegister(s.metrics.HistoryArchiveStatsCounter)
	s.ledgerBackend = ledgerbackend.WithMetrics(s.ledgerBackend, registry, "horizon")
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
// try to acquire a lock on `LastLedgerIngest value in key value store and only
// one instance will be able to acquire it. This happens in both initial processing
// and ledger processing. So this solves 3a and 3b in both 1a and 1b.
//
// Finally, 1a and 1b are tricky because we need to keep the latest version
// of order book graph in memory of each Horizon instance. To solve this:
// * For state init:
//   - If instance is a leader, we update the order book graph by running state
//     pipeline normally.
//   - If instance is NOT a leader, we build a graph from offers present in a
//     database. We completely omit state pipeline in this case.
//
// * For resuming:
//   - If instances is a leader, it runs full ledger pipeline, including updating
//     a database.
//   - If instances is a NOT leader, it runs ledger pipeline without updating a
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
	s.ledgerBackend = &fakeLedgerBackend{
		numTransactions:       numTransactions,
		changesPerTransaction: changesPerTransaction,
	}
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

// BuildState runs the state ingestion on selected checkpoint ledger then exits.
// When skipChecks is true it skips bucket list hash verification and protocol version check.
func (s *system) BuildState(sequence uint32, skipChecks bool) error {
	return s.runStateMachine(buildState{
		checkpointLedger: sequence,
		skipChecks:       skipChecks,
		stop:             true,
	})
}

func validateRanges(ledgerRanges []history.LedgerRange) error {
	for i, cur := range ledgerRanges {
		if cur.StartSequence > cur.EndSequence {
			return errors.Errorf("Invalid range: %v from > to", cur)
		}
		if cur.StartSequence == 0 {
			return errors.Errorf("Invalid range: %v genesis ledger starts at 1", cur)
		}
		if i == 0 {
			continue
		}
		prev := ledgerRanges[i-1]
		if prev.EndSequence >= cur.StartSequence {
			return errors.Errorf("ranges are not sorted prevRange %v curRange %v", prev, cur)
		}
	}
	return nil
}

// ReingestRange runs the ingestion pipeline on the range of ledgers ingesting
// history data only.
func (s *system) ReingestRange(ledgerRanges []history.LedgerRange, force bool, rebuildTradeAgg bool) error {
	if err := validateRanges(ledgerRanges); err != nil {
		return err
	}
	for _, cur := range ledgerRanges {
		run := func() error {
			return s.runStateMachine(reingestHistoryRangeState{
				fromLedger: cur.StartSequence,
				toLedger:   cur.EndSequence,
				force:      force,
			})
		}
		err := run()
		for retry := 0; err != nil && retry < s.maxReingestRetries; retry++ {
			log.Warnf("reingest range [%d, %d] failed (%s), retrying", cur.StartSequence, cur.EndSequence, err.Error())
			time.Sleep(time.Second * time.Duration(s.reingestRetryBackoffSeconds))
			err = run()
		}
		if err != nil {
			return err
		}
		if rebuildTradeAgg {
			err = s.RebuildTradeAggregationBuckets(cur.StartSequence, cur.EndSequence)
			if err != nil {
				return errors.Wrap(err, "Error rebuilding trade aggregations")
			}
		}
	}
	return nil
}

func (s *system) RebuildTradeAggregationBuckets(fromLedger, toLedger uint32) error {
	return s.historyQ.RebuildTradeAggregationBuckets(s.ctx, fromLedger, toLedger, s.config.RoundingSlippageFilter)
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

		s.currentStateMutex.Lock()
		s.currentState = cur.GetState()
		s.currentStateMutex.Unlock()

		next, err := cur.run(s)
		if err != nil {
			logger := log.WithFields(logpkg.F{
				"error":         err,
				"current_state": cur,
				"next_state":    next.node,
			})
			if isCancelledError(s.ctx, err) {
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

func (s *system) maybeVerifyState(lastIngestedLedger uint32, expectedBucketListHash xdr.Hash) {
	stateInvalid, err := s.historyQ.GetExpStateInvalid(s.ctx)
	if err != nil {
		if !isCancelledError(s.ctx, err) {
			log.WithField("err", err).Error("Error getting state invalid value")
		}
		return
	}

	// Run verification routine only when...
	if !stateInvalid && // state has not been proved to be invalid...
		!s.disableStateVerification && // state verification is not disabled...
		s.runStateVerificationOnLedger(lastIngestedLedger) { // it's a ledger eligible for state verification.
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			err := s.verifyState(true, lastIngestedLedger, expectedBucketListHash)
			if err != nil {
				if isCancelledError(s.ctx, err) {
					return
				}

				errorCount := s.incrementStateVerificationErrors()
				switch errors.Cause(err).(type) {
				case ingest.StateError:
					markStateInvalid(s.ctx, s.historyQ, err)
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

func (s *system) maybeReapLookupTables(lastIngestedLedger uint32) {
	if !s.config.EnableReapLookupTables {
		return
	}

	// Check if lastIngestedLedger is the last one available in the backend
	sequence, err := s.ledgerBackend.GetLatestLedgerSequence(s.ctx)
	if err != nil {
		log.WithField("err", err).Error("Error getting latest ledger sequence from backend")
		return
	}

	if sequence != lastIngestedLedger {
		// Catching up - skip reaping tables in this cycle.
		return
	}

	err = s.historyQ.Begin(s.ctx)
	if err != nil {
		log.WithField("err", err).Error("Error starting a transaction")
		return
	}
	defer s.historyQ.Rollback()

	// If so block ingestion in the cluster to reap tables
	_, err = s.historyQ.GetLastLedgerIngest(s.ctx)
	if err != nil {
		log.WithField("err", err).Error(getLastIngestedErrMsg)
		return
	}

	// Make sure reaping will not take more than 5s, which is average ledger
	// closing time.
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	reapStart := time.Now()
	deletedCount, newOffsets, err := s.historyQ.ReapLookupTables(ctx, s.reapOffsets)
	if err != nil {
		log.WithField("err", err).Warn("Error reaping lookup tables")
		return
	}

	err = s.historyQ.Commit()
	if err != nil {
		log.WithField("err", err).Error("Error committing a transaction")
		return
	}

	totalDeleted := int64(0)
	reapLog := log
	for table, c := range deletedCount {
		totalDeleted += c
		reapLog = reapLog.WithField(table, c)
	}

	if totalDeleted > 0 {
		reapLog.Info("Reaper deleted rows from lookup tables")
	}

	s.reapOffsets = newOffsets
	reapDuration := time.Since(reapStart).Seconds()
	s.Metrics().LedgerIngestionReapLookupTablesDuration.Observe(float64(reapDuration))
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
	s.historyQ.Close()
	if err := s.ledgerBackend.Close(); err != nil {
		log.WithError(err).Info("could not close ledger backend")
	}
}

func markStateInvalid(ctx context.Context, historyQ history.IngestionQ, err error) {
	log.WithField("err", err).Error("STATE IS INVALID!")
	q := historyQ.CloneIngestionQ()
	if err := q.UpdateExpStateInvalid(ctx, true); err != nil {
		log.WithField("err", err).Error(updateExpStateInvalidErrMsg)
	}
}

func isCancelledError(ctx context.Context, err error) bool {
	cause := errors.Cause(err)
	return cause == context.Canceled || cause == db.ErrCancelled ||
		(ctx.Err() == context.Canceled && cause == db.ErrAlreadyRolledback)
}
