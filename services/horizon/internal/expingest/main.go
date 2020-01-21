// Package expingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package expingest

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
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
)

var log = logpkg.DefaultLogger.WithField("service", "expingest")

type Config struct {
	CoreSession       *db.Session
	StellarCoreURL    string
	NetworkPassphrase string

	HistorySession           *db.Session
	HistoryArchiveURL        string
	TempSet                  io.TempSet
	DisableStateVerification bool

	// MaxStreamRetries determines how many times the reader will retry when encountering
	// errors while streaming xdr bucket entries from the history archive.
	// Set MaxStreamRetries to 0 if there should be no retry attempts
	MaxStreamRetries int

	OrderBookGraph           *orderbook.OrderBookGraph
	IngestFailedTransactions bool
}

type dbQ interface {
	Begin() error
	Commit() error
	Clone() *db.Session
	Rollback() error
	GetTx() *sqlx.Tx
	GetLastLedgerExpIngest() (uint32, error)
	GetExpIngestVersion() (int, error)
	UpdateLastLedgerExpIngest(uint32) error
	UpdateExpStateInvalid(bool) error
	UpdateExpIngestVersion(int) error
	GetExpStateInvalid() (bool, error)
	GetAllOffers() ([]history.Offer, error)
	GetLatestLedger() (uint32, error)
	TruncateExpingestStateTables() error
	DeleteRangeAll(start, end int64) error
}

type dbSession interface {
	Clone() *db.Session
}

type liveSession interface {
	Run() error
	RunFromCheckpoint(checkpointLedger uint32) error
	GetArchive() historyarchive.ArchiveInterface
	Resume(ledgerSequence uint32) error
	GetLatestSuccessfullyProcessedLedger() (ledgerSequence uint32, processed bool)
	Shutdown()
}

type systemState string

const (
	initState                         systemState = "init"
	ingestHistoryRangeState           systemState = "ingestHistoryRange"
	waitForCheckpointState            systemState = "waitForCheckpoint"
	loadOffersIntoMemoryState         systemState = "loadOffersIntoMemory"
	buildStateAndResumeIngestionState systemState = "buildStateAndResumeIngestion"
	resumeState                       systemState = "resume"
	verifyRangeState                  systemState = "verifyRange"
	shutdownState                     systemState = "shutdown"
)

type state struct {
	systemState                       systemState
	latestSuccessfullyProcessedLedger uint32

	checkpointLedger uint32

	rangeFromLedger   uint32
	rangeToLedger     uint32
	rangeVerifyState  bool
	rangeClearHistory bool

	shutdownWhenDone bool

	returnError error
}

type System struct {
	config           Config
	state            state
	session          liveSession
	rangeSession     *ingest.RangeSession
	historyQ         dbQ
	historySession   dbSession
	historyAdapter   adapters.HistoryArchiveAdapterInterface
	graph            *orderbook.OrderBookGraph
	maxStreamRetries int
	wg               sync.WaitGroup
	shutdown         chan struct{}

	// stateVerificationRunning is true when verification routine is currently
	// running.
	stateVerificationMutex sync.Mutex
	// number of consecutive state verification runs which encountered errors
	stateVerificationErrors  int
	stateVerificationRunning bool
	disableStateVerification bool
}

func NewSystem(config Config) (*System, error) {
	archive, err := createArchive(config.HistoryArchiveURL)
	if err != nil {
		return nil, errors.Wrap(err, "error creating history archive")
	}

	ledgerBackend, err := ledgerbackend.NewDatabaseBackendFromSession(config.CoreSession)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ledger backend")
	}

	// Make historySession synchronized so it can be used in the pipeline
	// (saving to DB in multiple goroutines at the same time).
	historySession := config.HistorySession.Clone()
	historySession.Synchronized = true

	historyQ := &history.Q{historySession}

	statePipeline := buildStatePipeline(historyQ, config.OrderBookGraph)
	ledgerPipeline := buildLedgerPipeline(
		historyQ,
		config.OrderBookGraph,
		config.IngestFailedTransactions,
	)

	session := &ingest.LiveSession{
		Archive:          archive,
		MaxStreamRetries: config.MaxStreamRetries,
		LedgerBackend:    ledgerBackend,
		StatePipeline:    statePipeline,
		LedgerPipeline:   ledgerPipeline,
		StellarCoreClient: &stellarcore.Client{
			URL: config.StellarCoreURL,
		},

		StateReporter:  &LoggingStateReporter{Log: log, Interval: 100000},
		LedgerReporter: &LoggingLedgerReporter{Log: log},

		TempSet: config.TempSet,
	}

	rangeSession := &ingest.RangeSession{
		Archive:           archive,
		MaxStreamRetries:  config.MaxStreamRetries,
		LedgerBackend:     ledgerBackend,
		StatePipeline:     statePipeline,
		LedgerPipeline:    ledgerPipeline,
		NetworkPassphrase: config.NetworkPassphrase,

		StateReporter:  &LoggingStateReporter{Log: log, Interval: 100000},
		LedgerReporter: &LoggingLedgerReporter{Log: log},

		TempSet: config.TempSet,
	}

	system := &System{
		config:                   config,
		session:                  session,
		rangeSession:             rangeSession,
		historySession:           historySession,
		historyAdapter:           adapters.MakeHistoryArchiveAdapter(session.GetArchive()),
		historyQ:                 historyQ,
		graph:                    config.OrderBookGraph,
		disableStateVerification: config.DisableStateVerification,
		maxStreamRetries:         config.MaxStreamRetries,
	}

	addPipelineHooks(
		system,
		session.StatePipeline,
		historySession,
		session,
		config.OrderBookGraph,
	)
	addPipelineHooks(
		system,
		session.LedgerPipeline,
		historySession,
		session,
		config.OrderBookGraph,
	)

	return system, nil
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
func (s *System) Run() {
	s.state = state{systemState: initState}
	s.run()
}

// VerifyRange runs the ingestion pipeline on the range of ledgers. When
// verifyState is true it verifies the state when ingestion is complete.
func (s *System) VerifyRange(fromLedger, toLedger uint32, verifyState bool) error {
	s.state = state{
		systemState:      verifyRangeState,
		rangeFromLedger:  fromLedger,
		rangeToLedger:    toLedger,
		rangeVerifyState: verifyState,
	}
	return s.run()
}

// ReingestRange runs the ingestion pipeline on the range of ledgers ingesting
// history data only.
func (s *System) ReingestRange(fromLedger, toLedger uint32) error {
	s.state = state{
		systemState:       ingestHistoryRangeState,
		rangeFromLedger:   fromLedger,
		rangeToLedger:     toLedger,
		rangeClearHistory: true,
		shutdownWhenDone:  true,
	}
	return s.run()
}

func (s *System) run() error {
	s.shutdown = make(chan struct{})
	// Expingest is an experimental package so we don't want entire Horizon app
	// to crash in case of unexpected errors.
	// TODO: This should be removed when expingest is no longer experimental.
	defer func() {
		s.wg.Wait()
		if r := recover(); r != nil {
			log.WithFields(logpkg.F{
				"err":   r,
				"stack": string(debug.Stack()),
			}).Error("expingest panic")
		}
	}()

	log.WithFields(logpkg.F{"current_state": s.state}).Info("Ingestion system initial state")

	for {
		nextState, err := s.runCurrentState()
		if err != nil {
			log.WithFields(logpkg.F{
				"error":         err,
				"current_state": s.state,
				"next_state":    nextState,
			}).Error("Error in ingestion state machine")
		}

		// Exit after processing shutdownState
		if s.state.systemState == shutdownState {
			return s.state.returnError
		}

		select {
		case <-s.shutdown:
			log.Info("Received shut down signal...")
			nextState = state{systemState: shutdownState}
		case <-time.After(time.Second):
		}

		log.WithFields(logpkg.F{
			"current_state": s.state,
			"next_state":    nextState,
		}).Info("Ingestion system state machine transition")

		s.state = nextState
	}
}

func (s *System) runCurrentState() (state, error) {
	// Transaction will be commited or rolled back in pipelines post hooks
	// or below in case of errors.
	if tx := s.historyQ.GetTx(); tx == nil {
		err := s.historyQ.Begin()
		if err != nil {
			return state{systemState: initState}, errors.Wrap(err, "Error in Begin")
		}
	}

	var nextState state
	var err error

	switch s.state.systemState {
	case initState:
		nextState, err = s.init()
	case ingestHistoryRangeState:
		nextState, err = s.ingestHistoryRange()
	case waitForCheckpointState:
		nextState, err = s.waitForCheckpoint()
	case loadOffersIntoMemoryState:
		nextState, err = s.loadOffersIntoMemory()
	case buildStateAndResumeIngestionState:
		nextState, err = s.buildStateAndResumeIngestion()
	case resumeState:
		nextState, err = s.resume()
	case verifyRangeState:
		nextState, err = s.verifyRange()
	case shutdownState:
		s.historyQ.Rollback()
		log.Info("Shut down")
		nextState, err = s.state, nil
	default:
		panic(fmt.Sprintf("Unknown state %+v", s.state.systemState))
	}

	if err != nil {
		// We rollback in pipelines post-hooks but the error can happen before
		// pipeline starts processing.
		s.historyQ.Rollback()
	}

	return nextState, err
}

func (s *System) init() (state, error) {
	// This will get the value `FOR UPDATE`, blocking it for other nodes.
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error getting last ingested ledger")
	}

	ingestVersion, err := s.historyQ.GetExpIngestVersion()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error getting exp ingest version")
	}

	lastHistoryLedger, err := s.historyQ.GetLatestLedger()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error getting last history ledger sequence")
	}

	if ingestVersion != CurrentVersion || lastIngestedLedger == 0 {
		// This block is either starting from empty state or ingestion
		// version upgrade.
		// This will always run on a single instance due to the fact that
		// `LastLedgerExpIngest` value is blocked for update and will always
		// be updated when leading instance finishes processing state.
		// In case of errors it will start `Init` from the beginning.
		log.Info("Starting ingestion system from empty state...")

		if lastHistoryLedger != 0 {
			// There are ledgers in history_ledgers table. This means that the
			// old or new ingest system was running prior the upgrade. In both
			// cases we need to:
			// * Wait for the checkpoint ledger if the latest history ledger is
			//   greater that the latest checkpoint ledger.
			// * Catchup history data if the latest history ledger is less than
			//   the latest checkpoint ledger.
			// * Build state from the last checkpoint if the latest history ledger
			//   is equal to the latest checkpoint ledger.
			lastCheckpoint, err := s.historyAdapter.GetLatestLedgerSequence()
			if err != nil {
				return state{systemState: initState}, errors.Wrap(err, "Error getting last checkpoint")
			}

			switch {
			case lastHistoryLedger > lastCheckpoint:
				return state{systemState: waitForCheckpointState}, nil
			case lastHistoryLedger < lastCheckpoint:
				return state{
					systemState:     ingestHistoryRangeState,
					rangeFromLedger: lastHistoryLedger + 1,
					rangeToLedger:   lastCheckpoint,
				}, nil
			default: // lastHistoryLedger == lastCheckpoint
				// Build state but make sure it's using `lastCheckpoint`. It's possible
				// that the new checkpoint will be created during state transition.
				return state{
					systemState:      buildStateAndResumeIngestionState,
					checkpointLedger: lastCheckpoint,
				}, nil
			}
		}

		return state{
			systemState:      buildStateAndResumeIngestionState,
			checkpointLedger: 0,
		}, nil
	}

	switch {
	case lastHistoryLedger > lastIngestedLedger:
		// Expingest was running at some point the past but was turned off.
		// Now it's on by default but the latest history ledger is greater
		// than the latest expingest ledger. We reset the exp ledger sequence
		// so init state will rebuild the state correctly.
		err := s.historyQ.UpdateLastLedgerExpIngest(0)
		if err != nil {
			return state{systemState: initState}, errors.Wrap(err, "Error updating last ingested ledger")
		}
		err = s.historyQ.Commit()
		if err != nil {
			return state{systemState: initState}, errors.Wrap(err, "Error updating last ingested ledger")
		}
		return state{systemState: initState}, nil
	case lastHistoryLedger < lastIngestedLedger:
		// Expingest was running at some point the past but was turned off.
		// Now it's on by default but the latest history ledger is less
		// than the latest expingest ledger. We catchup history.
		return state{
			systemState:     ingestHistoryRangeState,
			rangeFromLedger: lastHistoryLedger + 1,
			rangeToLedger:   lastIngestedLedger,
		}, nil
	default: // lastHistoryLedger == lastIngestedLedger
		// The other node already ingested a state (just now or in the past)
		// so we need to get offers from a DB, then resume session normally.
		// State pipeline is NOT processed.
		log.WithField("last_ledger", lastIngestedLedger).
			Info("Resuming ingestion system from last processed ledger...")

		return state{
			systemState:                       loadOffersIntoMemoryState,
			latestSuccessfullyProcessedLedger: lastIngestedLedger,
		}, nil
	}
}

// loadOffersIntoMemory loads offers into memory. If successful, it changes the
// state to `resumeState`. In case of errors it always changes the state to
// `init` because state function returning errors rollback DB transaction.
func (s *System) loadOffersIntoMemory() (state, error) {
	defer s.graph.Discard()

	log.Info("Loading offers from a database into memory store...")
	start := time.Now()

	offers, err := s.historyQ.GetAllOffers()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "GetAllOffers error")
	}

	for _, offer := range offers {
		sellerID := xdr.MustAddress(offer.SellerID)
		s.graph.AddOffer(xdr.OfferEntry{
			SellerId: sellerID,
			OfferId:  offer.OfferID,
			Selling:  offer.SellingAsset,
			Buying:   offer.BuyingAsset,
			Amount:   offer.Amount,
			Price: xdr.Price{
				N: xdr.Int32(offer.Pricen),
				D: xdr.Int32(offer.Priced),
			},
			Flags: xdr.Uint32(offer.Flags),
		})
	}

	err = s.graph.Apply(s.state.latestSuccessfullyProcessedLedger)
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error running graph.Apply")
	}

	log.WithField(
		"duration",
		time.Since(start).Seconds(),
	).Info("Finished loading offers from a database into memory store")

	return state{
		systemState:                       resumeState,
		latestSuccessfullyProcessedLedger: s.state.latestSuccessfullyProcessedLedger,
	}, nil
}

func (s *System) buildStateAndResumeIngestion() (state, error) {
	// Clear last_ingested_ledger in key value store
	err := s.historyQ.UpdateLastLedgerExpIngest(0)
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error updating last ingested ledger")
	}

	// Clear invalid state in key value store. It's possible that upgraded
	// ingestion is fixing it.
	err = s.historyQ.UpdateExpStateInvalid(false)
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error updating state invalid value")
	}

	err = s.historyQ.TruncateExpingestStateTables()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error clearing ingest tables")
	}

	if s.state.checkpointLedger != 0 {
		err = s.session.RunFromCheckpoint(s.state.checkpointLedger)
	} else {
		err = s.session.Run()
	}
	if err != nil {
		// Check if session processed a state, if so, continue since the
		// last processed ledger, otherwise start over.
		latestSuccessfullyProcessedLedger, processed := s.session.GetLatestSuccessfullyProcessedLedger()
		if !processed {
			return state{systemState: initState}, err
		}

		return state{
			systemState:                       resumeState,
			latestSuccessfullyProcessedLedger: latestSuccessfullyProcessedLedger,
		}, err
	}

	return state{systemState: shutdownState}, nil
}

func (s *System) resume() (state, error) {
	err := s.session.Resume(s.state.latestSuccessfullyProcessedLedger + 1)
	if err != nil {
		err = errors.Wrap(err, "Error returned from ingest.LiveSession")
		// If no ledgers processed so far, try again with the
		// latestSuccessfullyProcessedLedger+1 (do nothing).
		// Otherwise, set latestSuccessfullyProcessedLedger to the last
		// successfully ingested ledger in the machine state.
		sessionLastLedger, processed := s.session.GetLatestSuccessfullyProcessedLedger()
		if processed {
			return state{
				systemState:                       resumeState,
				latestSuccessfullyProcessedLedger: sessionLastLedger,
			}, err
		}

		return s.state, err
	}

	return state{systemState: shutdownState}, nil
}

// ingestHistoryRange is used when catching up history data and when reingesting
// range.
func (s *System) ingestHistoryRange() (state, error) {
	returnState := initState
	if s.state.shutdownWhenDone {
		// Shutdown when done - used in `reingest range` command.
		returnState = shutdownState
	}

	if s.state.rangeClearHistory {
		// Clear history data before ingesting - used in `reingest range` command.
		var start, end int64
		if s.state.rangeFromLedger == 1 {
			start = 0
		} else {
			start = toid.New(int32(s.state.rangeFromLedger), 0, 0).ToInt64()
		}

		// In DeleteRangeAll `end` is exclusive so we add 1 to remove the last ledger
		// too.
		if s.state.rangeToLedger == 1 {
			end = toid.New(1, 0, 0).ToInt64()
		} else {
			end = toid.New(int32(s.state.rangeToLedger+1), 0, 0).ToInt64()
		}

		err := s.historyQ.DeleteRangeAll(start, end)
		if err != nil {
			return state{systemState: returnState}, err
		}
	}

	statePipeline := s.rangeSession.StatePipeline
	s.rangeSession.StatePipeline = nil
	// -1 here because RangeSession is ingesting state at FromLedger and then
	// continues with ingestion from FromLedger + 1
	s.rangeSession.FromLedger = s.state.rangeFromLedger - 1
	s.rangeSession.ToLedger = s.state.rangeToLedger
	// Temporarily disable disableStateVerification
	s.disableStateVerification = true
	defer func() {
		// Revert previous values
		s.rangeSession.StatePipeline = statePipeline
		s.disableStateVerification = s.config.DisableStateVerification
	}()

	err := s.rangeSession.Run()
	if err != nil {
		return state{systemState: returnState}, err
	}

	err = s.historyQ.Commit()
	if err != nil {
		return state{systemState: returnState}, err
	}

	return state{systemState: returnState}, nil
}

func (s *System) waitForCheckpoint() (state, error) {
	log.Info("Waiting for the next checkpoint...")
	time.Sleep(10 * time.Second)
	return state{systemState: initState}, nil
}

func (s *System) verifyRange() (state, error) {
	// Simple check if DB clean
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		err = errors.Wrap(err, "Error getting last ledger")
		return state{systemState: shutdownState, returnError: err}, err
	}

	if lastIngestedLedger != 0 {
		return state{systemState: shutdownState, returnError: errors.New("Database not empty")}, err
	}

	s.rangeSession.FromLedger = s.state.rangeFromLedger
	s.rangeSession.ToLedger = s.state.rangeToLedger
	// It's fine to change System settings because the next state of verifyRange
	// is always shut down.
	s.disableStateVerification = true

	err = s.rangeSession.Run()
	if err == nil {
		if s.state.rangeVerifyState {
			err = s.verifyState(s.graph.OffersMap())
		}
	}

	return state{systemState: shutdownState, returnError: err}, err
}

func (s *System) incrementStateVerificationErrors() int {
	s.stateVerificationMutex.Lock()
	defer s.stateVerificationMutex.Unlock()
	s.stateVerificationErrors++
	return s.stateVerificationErrors
}

func (s *System) resetStateVerificationErrors() {
	s.stateVerificationMutex.Lock()
	defer s.stateVerificationMutex.Unlock()
	s.stateVerificationErrors = 0
}

func (s *System) Shutdown() {
	log.Info("Shutting down ingestion system...")
	s.session.Shutdown()
	if s.rangeSession != nil {
		s.rangeSession.Shutdown()
	}
	s.stateVerificationMutex.Lock()
	defer s.stateVerificationMutex.Unlock()
	if s.stateVerificationRunning {
		log.Info("Shutting down state verifier...")
	}
	close(s.shutdown)
}

func createArchive(archiveURL string) (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{},
	)
}
