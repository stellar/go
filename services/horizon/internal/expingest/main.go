// Package expingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package expingest

import (
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/exp/ingest/adapters"
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
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

	stateVerificationErrorThreshold = 3
)

var log = logpkg.DefaultLogger.WithField("service", "expingest")

type Config struct {
	CoreSession       *db.Session
	StellarCoreURL    string
	NetworkPassphrase string

	HistorySession           *db.Session
	HistoryArchiveURL        string
	DisableStateVerification bool

	// MaxStreamRetries determines how many times the reader will retry when encountering
	// errors while streaming xdr bucket entries from the history archive.
	// Set MaxStreamRetries to 0 if there should be no retry attempts
	MaxStreamRetries int

	OrderBookGraph           *orderbook.OrderBookGraph
	IngestFailedTransactions bool
}

type dbQ interface {
	history.QAccounts
	history.QAssetStats
	history.QData
	history.QEffects
	history.QLedgers
	history.QOffers
	history.QOperations
	// QParticipants
	// Copy the small interfaces with shared methods directly, otherwise error:
	// duplicate method CreateAccounts
	NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) history.TransactionParticipantsBatchInsertBuilder
	NewOperationParticipantBatchInsertBuilder(maxBatchSize int) history.OperationParticipantBatchInsertBuilder
	history.QSigners
	//QTrades
	NewTradeBatchInsertBuilder(maxBatchSize int) history.TradeBatchInsertBuilder
	CreateAssets(assets []xdr.Asset) (map[string]history.Asset, error)
	history.QTransactions
	history.QTrustLines

	Begin() error
	Commit() error
	Clone() *db.Session
	Rollback() error
	GetTx() *sqlx.Tx
	GetExpIngestVersion() (int, error)
	UpdateExpStateInvalid(bool) error
	UpdateExpIngestVersion(int) error
	GetExpStateInvalid() (bool, error)
	GetLatestLedger() (uint32, error)
	TruncateExpingestStateTables() error
	DeleteRangeAll(start, end int64) error
}

type systemState string

const (
	initState               systemState = "init"
	ingestHistoryRangeState systemState = "ingestHistoryRange"
	waitForCheckpointState  systemState = "waitForCheckpoint"
	buildStateState         systemState = "buildState"
	resumeState             systemState = "resume"
	verifyRangeState        systemState = "verifyRange"
	shutdownState           systemState = "shutdown"
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
	config Config
	state  state

	graph    *orderbook.OrderBookGraph
	historyQ dbQ

	historyArchive *historyarchive.Archive
	ledgerBackend  *ledgerbackend.DatabaseBackend
	historyAdapter adapters.HistoryArchiveAdapterInterface

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

	historyQ := &history.Q{config.HistorySession.Clone()}

	system := &System{
		historyArchive:           archive,
		historyAdapter:           adapters.MakeHistoryArchiveAdapter(archive),
		ledgerBackend:            ledgerBackend,
		config:                   config,
		historyQ:                 historyQ,
		graph:                    config.OrderBookGraph,
		disableStateVerification: config.DisableStateVerification,
		maxStreamRetries:         config.MaxStreamRetries,
	}

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
	defer func() {
		s.wg.Wait()
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
	// Every node in the state machine is responsible for
	// creating and disposing its own transaction.
	// We should never enter a new state with the transaction
	// from the previous state.
	if s.historyQ.GetTx() != nil {
		panic("unexpected transaction")
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
	case buildStateState:
		nextState, err = s.buildState()
	case resumeState:
		nextState, err = s.resume()
	case verifyRangeState:
		nextState, err = s.verifyRange()
	case shutdownState:
		log.Info("Shut down")
		nextState, err = s.state, nil
	default:
		panic(fmt.Sprintf("Unknown state %+v", s.state.systemState))
	}

	return nextState, err
}

func (s *System) init() (state, error) {
	if err := s.historyQ.Begin(); err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()

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
		var lastCheckpoint uint32
		lastCheckpoint, err = s.historyAdapter.GetLatestLedgerSequence()
		if err != nil {
			return state{systemState: initState}, errors.Wrap(err, "Error getting last checkpoint")
		}

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
					systemState:      buildStateState,
					checkpointLedger: lastCheckpoint,
				}, nil
			}
		}

		return state{
			systemState:      buildStateState,
			checkpointLedger: lastCheckpoint,
		}, nil
	}

	switch {
	case lastHistoryLedger > lastIngestedLedger:
		// Expingest was running at some point the past but was turned off.
		// Now it's on by default but the latest history ledger is greater
		// than the latest expingest ledger. We reset the exp ledger sequence
		// so init state will rebuild the state correctly.
		err = s.historyQ.UpdateLastLedgerExpIngest(0)
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

		if err = s.loadOffersIntoMemory(); err != nil {
			return state{systemState: initState},
				errors.Wrap(err, "Error loading offers into in memory graph")
		}

		return state{
			systemState:                       resumeState,
			latestSuccessfullyProcessedLedger: lastIngestedLedger,
		}, nil
	}
}

func (s *System) loadOffersIntoMemory() error {
	defer s.graph.Discard()

	log.Info("Loading offers from a database into memory store...")
	start := time.Now()

	offers, err := s.historyQ.GetAllOffers()
	if err != nil {
		return errors.Wrap(err, "GetAllOffers error")
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
		return errors.Wrap(err, "Error running graph.Apply")
	}

	log.WithField(
		"duration",
		time.Since(start).Seconds(),
	).Info("Finished loading offers from a database into memory store")

	return nil
}

func (s *System) buildState() (state, error) {
	if s.state.checkpointLedger == 0 {
		return state{systemState: initState}, errors.New("unexpected checkpointLedger value")
	}

	if err := s.historyQ.Begin(); err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()
	defer s.graph.Discard()

	// We need to get this value `FOR UPDATE` so all other instances
	// are blocked.
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error getting last ledger")
	}

	ingestVersion, err := s.historyQ.GetExpIngestVersion()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error getting exp ingest version")
	}

	// Double check if we should proceed with state ingestion. It's possible that
	// another ingesting instance will be redirected to this state from `init`
	// but it's first to complete the task.
	if ingestVersion == CurrentVersion && lastIngestedLedger > 0 {
		log.Info("Another instance completed `buildState`. Skipping...")
		return state{systemState: initState}, nil
	}

	log.Info("Starting ingestion system from empty state...")

	// Clear last_ingested_ledger in key value store
	err = s.historyQ.UpdateLastLedgerExpIngest(0)
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

	log.WithFields(logpkg.F{
		"ledger": s.state.checkpointLedger,
	}).Info("Processing state")
	startTime := time.Now()

	err = s.runHistoryArchiveIngestion(s.state.checkpointLedger)
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error ingesting history archive")
	}

	if err = s.historyQ.UpdateLastLedgerExpIngest(s.state.checkpointLedger); err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error updating last ingested ledger")
	}

	if err = s.historyQ.UpdateExpIngestVersion(CurrentVersion); err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error updating expingest version")
	}

	err = s.historyQ.Commit()
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error commiting db transaction")
	}

	err = s.graph.Apply(s.state.checkpointLedger)
	if err != nil {
		return state{systemState: initState}, errors.Wrap(err, "Error applying order book changes")
	}

	log.WithFields(logpkg.F{
		"ledger":   s.state.checkpointLedger,
		"duration": time.Since(startTime).Seconds(),
	}).Info("Processed state")

	// If successful, continue from the next ledger
	return state{
		systemState:                       resumeState,
		latestSuccessfullyProcessedLedger: s.state.checkpointLedger,
	}, nil
}

func (s *System) resume() (state, error) {
	if s.state.latestSuccessfullyProcessedLedger == 0 {
		return state{systemState: initState}, errors.New("unexpected latestSuccessfullyProcessedLedger value")
	}

	if err := s.historyQ.Begin(); err != nil {
		return s.state,
			errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()
	defer s.graph.Discard()

	// This will get the value `FOR UPDATE`, blocking it for other nodes.
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return s.state,
			errors.Wrap(err, "Error getting last ingested ledger")
	}

	ingestLedger := s.state.latestSuccessfullyProcessedLedger + 1

	if ingestLedger > lastIngestedLedger+1 {
		return state{systemState: initState},
			errors.Wrap(
				err,
				"expected ingest ledger to be at most one greater"+
					"than last ingested ledger in db",
			)
	}

	log.WithField("sequence", ingestLedger).Info("Processing ledger")
	startTime := time.Now()

	if ingestLedger < lastIngestedLedger {
		// rollback because we will not be updating the DB
		// so there is no need to hold on to the distributed lock
		// and thereby block the other nodes from ingesting
		if err = s.historyQ.Rollback(); err != nil {
			return s.state,
				errors.Wrap(err, "Error rolling back transaction")
		}

		orderBookProcessor := s.buildOrderBookChangeProcessor()
		err = s.runChangeProcessorOnLedger(orderBookProcessor, ingestLedger)
		if err != nil {
			return s.state,
				errors.Wrap(err, "Error running change processor on ledger")

		}

		if err = s.graph.Apply(ingestLedger); err != nil {
			return s.state,
				errors.Wrap(err, "Error applying graph changes from ledger")
		}

		log.WithFields(logpkg.F{
			"sequence": ingestLedger,
			"duration": time.Since(startTime).Seconds(),
			"state":    false,
			"ledger":   false,
			"graph":    true,
			"commit":   false,
		}).Info("Processed ledger")

		return state{
			systemState:                       resumeState,
			latestSuccessfullyProcessedLedger: ingestLedger,
		}, nil
	}

	err = s.runAllProcessorsOnLedger(ingestLedger)
	if err != nil {
		return s.state,
			errors.Wrap(err, "Error running processors on ledger")
	}

	err = s.historyQ.UpdateLastLedgerExpIngest(ingestLedger)
	if err != nil {
		return s.state,
			errors.Wrap(err, "Error updating last ingested ledger")
	}

	if err = s.graph.Apply(ingestLedger); err != nil {
		return s.state,
			errors.Wrap(err, "Error applying graph changes from ledger")
	}

	if err = s.historyQ.Commit(); err != nil {
		return s.state,
			errors.Wrap(err, "Error commiting processor transaction")
	}

	log.WithFields(logpkg.F{
		"sequence": ingestLedger,
		"duration": time.Since(startTime).Seconds(),
		"state":    true,
		"ledger":   true,
		"graph":    true,
		"commit":   true,
	}).Info("Processed ledger")

	s.maybeVerifyState(ingestLedger)

	return state{
		systemState:                       resumeState,
		latestSuccessfullyProcessedLedger: ingestLedger,
	}, nil
}

func (s *System) maybeVerifyState(lastIngestedLedger uint32) {
	stateInvalid, err := s.historyQ.GetExpStateInvalid()
	if err != nil {
		log.WithField("err", err).Error("Error getting state invalid value")
	}

	// Run verification routine only when...
	if !stateInvalid && // state has not been proved to be invalid...
		!s.disableStateVerification && // state verification is not disabled...
		historyarchive.IsCheckpoint(lastIngestedLedger) { // it's a checkpoint ledger.
		s.wg.Add(1)
		go func(graphOffersMap map[xdr.Int64]xdr.OfferEntry) {
			defer s.wg.Done()

			err := s.verifyState(graphOffersMap)
			if err != nil {
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
		}(s.graph.OffersMap())
	}
}

// ingestHistoryRange is used when catching up history data and when reingesting
// range.
func (s *System) ingestHistoryRange() (state, error) {
	returnState := initState
	validateStartLedger := true
	// unless we're running the horizon reingest range command we should
	// always check that the start ledger is equal to the last ledger
	// in the db plus one
	if s.state.shutdownWhenDone {
		// Shutdown when done - used in `reingest range` command.
		returnState = shutdownState
		validateStartLedger = false
	}

	if err := s.historyQ.Begin(); err != nil {
		return state{systemState: returnState},
			errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()

	// acquire distributed lock so no one else can perform ingestion operations.
	if _, err := s.historyQ.GetLastLedgerExpIngest(); err != nil {
		return state{systemState: returnState},
			errors.Wrap(err, "Error getting last ingested ledger")
	}

	if s.state.rangeClearHistory {
		// Clear history data before ingesting - used in `reingest range` command.
		start, end, err := toid.LedgerRangeInclusive(
			int32(s.state.rangeFromLedger),
			int32(s.state.rangeToLedger),
		)

		if err != nil {
			return state{systemState: returnState},
				errors.Wrap(err, "Invalid range")
		}

		err = s.historyQ.DeleteRangeAll(start, end)
		if err != nil {
			return state{systemState: returnState}, err
		}
	}

	if validateStartLedger {
		lastHistoryLedger, err := s.historyQ.GetLatestLedger()
		if err != nil {
			return state{systemState: returnState},
				errors.Wrap(err, "could not get latest history ledger")
		}

		// We should be ingesting the ledger which occurs after
		// lastHistoryLedger. Otherwise, some other horizon node has
		// already completed the ingest history range operation and
		// we should go back to the init state
		if lastHistoryLedger != s.state.rangeFromLedger-1 {
			return state{systemState: returnState}, nil
		}
	}

	for cur := s.state.rangeFromLedger; cur <= s.state.rangeToLedger; cur++ {
		log.WithField("sequence", cur).Info("Processing ledger")
		startTime := time.Now()

		if err := s.runTransactionProcessorsOnLedger(cur); err != nil {
			return state{systemState: returnState}, err
		}

		log.WithFields(logpkg.F{
			"sequence": cur,
			"duration": time.Since(startTime).Seconds(),
			"state":    false,
			"ledger":   true,
			"graph":    false,
			"commit":   false,
		}).Info("Processed ledger")
	}

	if err := s.historyQ.Commit(); err != nil {
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
	if err := s.historyQ.Begin(); err != nil {
		err = errors.Wrap(err, "Error starting a transaction")
		return state{systemState: shutdownState, returnError: err}, err
	}
	defer s.historyQ.Rollback()
	defer s.graph.Discard()

	// Simple check if DB clean
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		err = errors.Wrap(err, "Error getting last ledger")
		return state{systemState: shutdownState, returnError: err}, err
	}

	if lastIngestedLedger != 0 {
		err = errors.New("Database not empty")
		return state{systemState: shutdownState, returnError: err}, err
	}

	log.WithFields(logpkg.F{
		"ledger": s.state.rangeFromLedger,
	}).Info("Processing state")
	startTime := time.Now()

	err = s.runHistoryArchiveIngestion(s.state.rangeFromLedger)
	if err != nil {
		err = errors.Wrap(err, "Error ingesting history archive")
		return state{systemState: shutdownState, returnError: err}, err
	}

	if err = s.historyQ.Commit(); err != nil {
		err = errors.Wrap(err, "Error commiting")
		return state{systemState: shutdownState, returnError: err}, err
	}

	err = s.graph.Apply(s.state.checkpointLedger)
	if err != nil {
		err = errors.Wrap(err, "Error applying order book changes")
		return state{systemState: shutdownState, returnError: err}, err
	}

	log.WithFields(logpkg.F{
		"ledger":   s.state.rangeFromLedger,
		"duration": time.Since(startTime).Seconds(),
	}).Info("Processed state")

	for sequence := s.state.rangeFromLedger + 1; sequence <= s.state.rangeToLedger; sequence++ {
		log.WithField("sequence", sequence).Info("Processing ledger")
		startTime := time.Now()

		if err = s.historyQ.Begin(); err != nil {
			err = errors.Wrap(err, "Error starting a transaction")
			return state{systemState: shutdownState, returnError: err}, err
		}

		err = s.runAllProcessorsOnLedger(sequence)
		if err != nil {
			err = errors.Wrap(err, "Error running processors on ledger")
			return state{systemState: shutdownState, returnError: err}, err
		}

		err = s.historyQ.UpdateLastLedgerExpIngest(sequence)
		if err != nil {
			err = errors.Wrap(err, "Error updating last ingested ledger")
			return state{systemState: shutdownState, returnError: err}, err
		}

		if err = s.graph.Apply(sequence); err != nil {
			err = errors.Wrap(err, "Error applying graph history archive changes")
			return state{systemState: shutdownState, returnError: err}, err
		}

		if err = s.historyQ.Commit(); err != nil {
			err = errors.Wrap(err, "Error commiting")
			return state{systemState: shutdownState, returnError: err}, err
		}

		log.WithFields(logpkg.F{
			"sequence": sequence,
			"duration": time.Since(startTime).Seconds(),
			"state":    true,
			"ledger":   true,
			"graph":    true,
			"commit":   true,
		}).Info("Processed ledger")
	}

	if s.state.rangeVerifyState {
		err = s.verifyState(s.graph.OffersMap())
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
	s.stateVerificationMutex.Lock()
	defer s.stateVerificationMutex.Unlock()
	if s.stateVerificationRunning {
		log.Info("Shutting down state verifier...")
	}
	close(s.shutdown)
}

func markStateInvalid(historyQ dbQ, err error) {
	log.WithField("err", err).Error("STATE IS INVALID!")
	q := &history.Q{historyQ.Clone()}
	if err := q.UpdateExpStateInvalid(true); err != nil {
		log.WithField("err", err).Error("Error updating state invalid value")
	}
}

func createArchive(archiveURL string) (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{},
	)
}
