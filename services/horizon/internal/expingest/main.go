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
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
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
	CurrentVersion = 2 // in version 2 we added the orderbook and offers processors
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
	stateNodes := []*supportPipeline.PipelineNode{
		accountForSignerStateNode(historyQ),
		orderBookDBStateNode(historyQ),
		orderBookGraphStateNode(config.OrderBookGraph),
	}
	ledgerNodes := []*supportPipeline.PipelineNode{
		accountForSignerLedgerNode(historyQ),
		orderBookDBLedgerNode(historyQ),
		orderBookGraphLedgerNode(config.OrderBookGraph),
	}

	session := &ingest.LiveSession{
		Archive:        archive,
		LedgerBackend:  ledgerBackend,
		StatePipeline:  buildStatePipeline(stateNodes),
		LedgerPipeline: buildLedgerPipeline(ledgerNodes),
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

func (s *System) Run() {
	var ingestVersion int
	var lastIngestedLedger uint32
	var err error
	for {
		lastIngestedLedger, err = s.historyQ.GetLastLedgerExpIngest()
		if err != nil {
			log.WithField("error", err).Error("Error getting last ingested ledger")
			time.Sleep(time.Second)
			continue
		}

		ingestVersion, err = s.historyQ.GetExpIngestVersion()
		if err != nil {
			log.WithField("error", err).Error("Error getting exp ingest version")
			time.Sleep(time.Second)
			continue
		}

		break
	}

	if ingestVersion != CurrentVersion || lastIngestedLedger == 0 {
		log.Info("Starting ingestion system from empty state...")

		for {
			err = s.historyQ.Session.TruncateTables(
				history.ExperimentalIngestionTables,
			)
			if err != nil {
				log.WithField("error", err).Error("Error clearing ingest tables")
				time.Sleep(time.Second)
				continue
			}
			break
		}

		s.runFromEmptyState()
		return
	} else {
		log.WithField("last_ledger", lastIngestedLedger).
			Info("Resuming ingestion system from last processed ledger...")

		for {
			lastIngestedLedger, err = loadOrderBookGraphFromDB(s.historyQ, s.graph)
			if err != nil {
				log.WithField("error", err).Error("Error loading order book graph from db")
				time.Sleep(time.Second)
				continue
			}
			break
		}

		s.resumeFromLedger(lastIngestedLedger)
		return
	}
}

func loadOrderBookGraphFromDB(historyQ *history.Q, graph *orderbook.OrderBookGraph) (uint32, error) {
	var lastIngestedLedger uint32
	err := historyQ.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  true,
	})
	if err != nil {
		return lastIngestedLedger, err
	}
	defer historyQ.Rollback()
	defer graph.Discard()

	lastIngestedLedger, err = historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return lastIngestedLedger, err
	}

	offers, err := historyQ.GetAllOffers()
	if err != nil {
		return lastIngestedLedger, err
	}

	err = historyQ.Commit()
	if err == nil {
		for _, offer := range offers {
			sellerID := xdr.AccountId{}
			if err = sellerID.SetAddress(offer.SellerID); err != nil {
				return lastIngestedLedger, err
			}
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
	}
	return lastIngestedLedger, err
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
