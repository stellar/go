// Package expingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package expingest

import (
	"database/sql"
	"time"

	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	ilog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const (
	// CurrentVersion reflects the latest version of the ingestion
	// algorithm. This value is stored in KV store and is used to decide
	// if there's a need to reprocess the ledger state or reingest data.
	//
	// Version history:
	// - 1: Initial version
	// - 2: We added the orderbook, offers processors and distributed
	//      ingestion.
	CurrentVersion = 2
)

var log = ilog.DefaultLogger.WithField("service", "expingest")

type Config struct {
	CoreSession       *db.Session
	HistorySession    *db.Session
	HistoryArchiveURL string
	StellarCoreURL    string
	OrderBookGraph    *orderbook.OrderBookGraph
}

type System struct {
	session  *ingest.LiveSession
	historyQ *history.Q
	graph    *orderbook.OrderBookGraph
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

	historyQ := &history.Q{config.HistorySession}

	session := &ingest.LiveSession{
		Archive:        archive,
		LedgerBackend:  ledgerBackend,
		StatePipeline:  buildStatePipeline(historyQ, config.OrderBookGraph),
		LedgerPipeline: buildLedgerPipeline(historyQ, config.OrderBookGraph),
		StellarCoreClient: &stellarcore.Client{
			URL: config.StellarCoreURL,
		},

		StateReporter:  &LoggingStateReporter{Log: log, Interval: 100000},
		LedgerReporter: &LoggingLedgerReporter{Log: log},
	}

	addPipelineHooks(
		session.StatePipeline,
		config.HistorySession,
		session,
		config.OrderBookGraph,
	)
	addPipelineHooks(
		session.LedgerPipeline,
		config.HistorySession,
		session,
		config.OrderBookGraph,
	)

	return &System{
		session:  session,
		historyQ: historyQ,
		graph:    config.OrderBookGraph,
	}, nil
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
func (s *System) Run() error {
	// Transaction will be commited or rolled back in pipelines post hooks.
	// This needs to be `REPEATABLE READ` isolation because offers are loaded
	// from a DB later on and we need a consistent view.
	err := s.historyQ.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return errors.Wrap(err, "Error starting a transaction")
	}

	// This will get the value `FOR UPDATE`, blocking it for other nodes.
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest(true)
	if err != nil {
		return errors.Wrap(err, "Error getting last ingested ledger")
	}

	ingestVersion, err := s.historyQ.GetExpIngestVersion()
	if err != nil {
		return errors.Wrap(err, "Error getting exp ingest version")
	}

	if ingestVersion != CurrentVersion || lastIngestedLedger == 0 {
		// This block is either starting from empty state or ingestion
		// version upgrade.
		// This will always run on a single instance due to the fact that
		// `LastLedgerExpIngest` value is blocked for update and will always
		// be updated when leading instance finishes processing state.
		log.Info("Starting ingestion system from empty state...")

		err = s.historyQ.Session.TruncateTables(
			history.ExperimentalIngestionTables,
		)
		if err != nil {
			return errors.Wrap(err, "Error clearing ingest tables")
		}

		s.runFromEmptyState()
	} else {
		// The other node already ingested a state (just now or in the past)
		// so we need to get offers from a DB, then resume session normally.
		// State pipeline is NOT processed.
		log.WithField("last_ledger", lastIngestedLedger).
			Info("Resuming ingestion system from last processed ledger...")

		err = loadOrderBookGraphFromDB(s.historyQ, s.graph)
		if err != nil {
			return errors.Wrap(err, "Error loading order book graph from db")
		}

		s.resumeFromLedger(lastIngestedLedger)
	}

	return nil
}

func loadOrderBookGraphFromDB(historyQ *history.Q, graph *orderbook.OrderBookGraph) error {
	defer graph.Discard()

	log.Info("Loading offers from a database into memory store...")
	start := time.Now()

	offers, err := historyQ.GetAllOffers()
	if err != nil {
		return err
	}

	for _, offer := range offers {
		sellerID := xdr.MustAddress(offer.SellerID)
		graph.AddOffer(xdr.OfferEntry{
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

	err = graph.Apply()
	if err == nil {
		log.WithField("duration", time.Since(start).Seconds()).Info("Finished offers from a database into memory store")
	}

	return err
}

func (s *System) runFromEmptyState() {
	for {
		lastIngestedLedger := s.session.GetLatestProcessedLedger()

		var err error
		if lastIngestedLedger == 0 {
			err = s.session.Run()
		} else {
			s.resumeFromLedger(lastIngestedLedger)
			return
		}

		if err != nil {
			log.WithField("error", err).Error("Error returned from ingest.LiveSession")
			time.Sleep(time.Second)
			continue
		}

		log.Info("Session shut down")
		break
	}
}

func (s *System) resumeFromLedger(lastIngestedLedger uint32) {
	for {
		err := s.session.Resume(lastIngestedLedger + 1)
		if err != nil {
			lastIngestedLedger = s.session.GetLatestProcessedLedger()
			log.WithField("error", err).Error("Error returned from ingest.LiveSession")
			time.Sleep(time.Second)
			continue
		}

		log.Info("Session shut down")
		break
	}
}

func (s *System) Shutdown() {
	log.Info("Shutting down ingestion system...")
	s.session.Shutdown()
}

func createArchive(archiveURL string) (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{},
	)
}
