package expingest

import (
	"database/sql"
	"sort"

	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// OrderBookStream updates an in memory graph to be consistent with
// offers in the Horizon DB. Any offers which are created, modified, or removed
// from the Horizon DB during ingestion will be applied to the in memory order book
// graph. OrderBookStream assumes that no other component will update the
// in memory graph. However, it is safe for other go routines to use the
// in memory graph for read operations.
type OrderBookStream struct {
	OrderBookGraph *orderbook.OrderBookGraph
	HistorySession *db.Session
	lastLedger     uint32
}

type ingestionStatus struct {
	HistoryConsistentWithState bool
	StateInvalid               bool
	LastIngestedLedger         uint32
	LastOfferCompactionLedger  uint32
}

func (o *OrderBookStream) isValid(q history.IngestionQ) (ingestionStatus, error) {
	var status ingestionStatus
	var err error

	status.StateInvalid, err = q.GetExpStateInvalid()
	if err != nil {
		return status, errors.Wrap(err, "error from GetExpStateInvalid")
	}

	var lastHistoryLedger uint32
	lastHistoryLedger, err = q.GetLatestLedger()
	if err != nil {
		return status, errors.Wrap(err, "error from GetLatestLedger")
	}
	status.LastIngestedLedger, err = q.GetLastLedgerExpIngestNonBlocking()
	if err != nil {
		return status, errors.Wrap(err, "error from GetLastLedgerExpIngestNonBlocking")
	}
	status.LastOfferCompactionLedger, err = q.GetOfferCompactionSequence()
	if err != nil {
		return status, errors.Wrap(err, "error from GetOfferCompactionSequence")
	}

	status.HistoryConsistentWithState = (status.LastIngestedLedger == lastHistoryLedger) ||
		// Running ingestion on an empty DB is a special case because we first ingest from the history archive.
		// Then, on the next iteration, we ingest TX Meta from Stellar Core. So there is a brief
		// period where there will not be any rows in the history_ledgers table but that is ok.
		(lastHistoryLedger == 0)
	return status, nil
}

func (o *OrderBookStream) update(q history.IngestionQ) ([]history.Offer, []xdr.Int64, error) {
	status, err := o.isValid(q)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error from isValid check")
	}

	reset := o.lastLedger == 0
	if status.StateInvalid || !status.HistoryConsistentWithState {
		log.WithField("status", status).Warn("ingestion state is invalid")
		reset = true
	} else if status.LastIngestedLedger < o.lastLedger {
		log.WithField("status", status).
			WithField("last_ledger", o.lastLedger).
			Warn("ingestion is behind order book last ledger")
		reset = true
	} else if o.lastLedger > 0 && o.lastLedger < status.LastOfferCompactionLedger {
		log.WithField("status", status).
			WithField("last_ledger", o.lastLedger).
			Warn("order book is behind the last offer compaction ledger")
		reset = true
	}

	if reset {
		o.OrderBookGraph.Clear()
		o.lastLedger = 0

		// wait until offers in horizon db is valid before populating order book graph
		if status.StateInvalid || !status.HistoryConsistentWithState {
			log.WithField("status", status).
				Info("waiting for ingestion to update offers table")
			return []history.Offer{}, []xdr.Int64{}, nil
		}

		defer o.OrderBookGraph.Discard()
		var offers []history.Offer
		offers, err = loadOffersIntoGraph(q, o.OrderBookGraph)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error from loadOffersIntoGraph")
		}

		if err = o.OrderBookGraph.Apply(status.LastIngestedLedger); err != nil {
			return nil, nil, err
		}

		o.lastLedger = status.LastIngestedLedger
		return offers, []xdr.Int64{}, nil
	}

	if status.LastIngestedLedger == o.lastLedger {
		return []history.Offer{}, []xdr.Int64{}, nil
	}

	defer o.OrderBookGraph.Discard()

	var updated, rows []history.Offer
	var removed []xdr.Int64
	rows, err = q.GetUpdatedOffers(o.lastLedger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error from GetUpdatedOffers")
	}
	for _, row := range rows {
		if row.Deleted {
			removed = append(removed, row.OfferID)
		} else {
			updated = append(updated, row)
		}
	}
	addOffersToGraph(updated, o.OrderBookGraph)

	for _, offerID := range removed {
		o.OrderBookGraph.RemoveOffer(offerID)
	}

	if err = o.OrderBookGraph.Apply(status.LastIngestedLedger); err != nil {
		return nil, nil, errors.Wrap(err, "could not apply changes to orderbook")
	}

	o.lastLedger = status.LastIngestedLedger
	return updated, removed, nil
}

func verifyUpdatedOffers(ledger uint32, fromDB []history.Offer, fromIngestion []xdr.OfferEntry) {
	sort.Slice(fromDB, func(i, j int) bool {
		return fromDB[i].OfferID < fromDB[j].OfferID
	})
	sort.Slice(fromIngestion, func(i, j int) bool {
		return fromIngestion[i].OfferId < fromIngestion[j].OfferId
	})
	mismatch := len(fromDB) != len(fromIngestion)
	if !mismatch {
		for i, offerRow := range fromDB {
			offerEntry := fromIngestion[i]
			if offerRow.OfferID != offerEntry.OfferId ||
				offerRow.Amount != offerEntry.Amount ||
				offerRow.Priced != int32(offerEntry.Price.D) ||
				offerRow.Pricen != int32(offerEntry.Price.N) ||
				!offerRow.BuyingAsset.Equals(offerEntry.Buying) ||
				!offerRow.SellingAsset.Equals(offerEntry.Selling) ||
				offerRow.SellerID != offerEntry.SellerId.Address() {
				mismatch = true
				break
			}
		}
	}
	if mismatch {
		log.WithField("fromDB", fromDB).
			WithField("fromIngestion", fromIngestion).
			WithField("sequence", ledger).
			Warn("offers from db does not match offers from ingestion")
	}
}

func verifyRemovedOffers(ledger uint32, fromDB []xdr.Int64, fromIngestion []xdr.Int64) {
	sort.Slice(fromDB, func(i, j int) bool {
		return fromDB[i] < fromDB[j]
	})
	sort.Slice(fromIngestion, func(i, j int) bool {
		return fromIngestion[i] < fromIngestion[j]
	})
	mismatch := len(fromDB) != len(fromIngestion)
	if !mismatch {
		for i, offerRow := range fromDB {
			if offerRow != fromIngestion[i] {
				mismatch = true
				break
			}
		}
	}
	if mismatch {
		log.WithField("fromDB", fromDB).
			WithField("fromIngestion", fromIngestion).
			WithField("sequence", ledger).
			Warn("offers from db does not match offers from ingestion")
	}
}

func (o *OrderBookStream) updateAndVerify(sequence uint32, q history.IngestionQ, graph orderbook.OBGraph) {
	dbUpdates, dbRemoved, err := o.update(q)
	if err != nil {
		log.WithError(err).WithField("sequence", sequence).Info("could not update order book ingester")
		return
	}
	ingestionUpdates, ingestionRemoved := graph.Pending()
	verifyUpdatedOffers(sequence, dbUpdates, ingestionUpdates)
	verifyRemovedOffers(sequence, dbRemoved, ingestionRemoved)
}

// Update will query the Horizon DB for updates and apply them to the in memory order book graph.
// After calling this function the the in memory order book graph should be consistent with the
// Horizon DB (assuming no error is returned).
func (o *OrderBookStream) Update() error {
	q := &history.Q{o.HistorySession}
	if err := q.BeginTx(&sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead}); err != nil {
		return errors.Wrap(err, "could not start repeatable read transaction")
	}
	defer q.Rollback()

	_, _, err := o.update(q)
	return err
}
