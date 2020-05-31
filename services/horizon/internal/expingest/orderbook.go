package expingest

import (
	"database/sql"
	"sort"

	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
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
	OrderBookGraph orderbook.OBGraph
	HistoryQ       history.IngestionQ
	lastLedger     uint32
}

type ingestionStatus struct {
	HistoryConsistentWithState bool
	StateInvalid               bool
	LastIngestedLedger         uint32
	LastOfferCompactionLedger  uint32
}

func (o *OrderBookStream) getIngestionStatus() (ingestionStatus, error) {
	var status ingestionStatus
	var err error

	status.StateInvalid, err = o.HistoryQ.GetExpStateInvalid()
	if err != nil {
		return status, errors.Wrap(err, "Error from GetExpStateInvalid")
	}

	var lastHistoryLedger uint32
	lastHistoryLedger, err = o.HistoryQ.GetLatestLedger()
	if err != nil {
		return status, errors.Wrap(err, "Error from GetLatestLedger")
	}
	status.LastIngestedLedger, err = o.HistoryQ.GetLastLedgerExpIngestNonBlocking()
	if err != nil {
		return status, errors.Wrap(err, "Error from GetLastLedgerExpIngestNonBlocking")
	}
	status.LastOfferCompactionLedger, err = o.HistoryQ.GetOfferCompactionSequence()
	if err != nil {
		return status, errors.Wrap(err, "Error from GetOfferCompactionSequence")
	}

	status.HistoryConsistentWithState = (status.LastIngestedLedger == lastHistoryLedger) ||
		// Running ingestion on an empty DB is a special case because we first ingest from the history archive.
		// Then, on the next iteration, we ingest TX Meta from Stellar Core. So there is a brief
		// period where there will not be any rows in the history_ledgers table but that is ok.
		(lastHistoryLedger == 0)
	return status, nil
}

func (o *OrderBookStream) update(status ingestionStatus) ([]history.Offer, []xdr.Int64, error) {
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
		offers, err := loadOffersIntoGraph(o.HistoryQ, o.OrderBookGraph)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Error from loadOffersIntoGraph")
		}

		if err = o.OrderBookGraph.Apply(status.LastIngestedLedger); err != nil {
			return nil, nil, errors.Wrap(err, "Error applying changes to order book")
		}

		o.lastLedger = status.LastIngestedLedger
		return offers, []xdr.Int64{}, nil
	}

	if status.LastIngestedLedger == o.lastLedger {
		return []history.Offer{}, []xdr.Int64{}, nil
	}

	defer o.OrderBookGraph.Discard()

	var updated []history.Offer
	var removed []xdr.Int64
	rows, err := o.HistoryQ.GetUpdatedOffers(o.lastLedger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error from GetUpdatedOffers")
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
		return nil, nil, errors.Wrap(err, "Error applying changes to order book")
	}

	o.lastLedger = status.LastIngestedLedger
	return updated, removed, nil
}

func (o *OrderBookStream) verifyGraph(ingestion orderbook.OBGraph) {
	offers := o.OrderBookGraph.Offers()
	ingestionOffers := ingestion.Offers()
	mismatch := len(offers) != len(ingestionOffers)

	if !mismatch {
		sort.Slice(offers, func(i, j int) bool {
			return offers[i].OfferId < offers[j].OfferId
		})
		sort.Slice(ingestionOffers, func(i, j int) bool {
			return ingestionOffers[i].OfferId < ingestionOffers[j].OfferId
		})

		offerBase64, err := xdr.MarshalBase64(offers)
		if err != nil {
			log.WithError(err).Error("could not serialize offers")
			return
		}
		ingestionOffersBase64, err := xdr.MarshalBase64(ingestionOffers)
		if err != nil {
			log.WithError(err).Error("could not serialize ingestion offers")
			return
		}
		mismatch = offerBase64 != ingestionOffersBase64
	}

	if mismatch {
		log.WithField("stream_offers", offers).
			WithField("ingestion_offers", ingestionOffers).
			Error("offers derived from order book stream does not match offers from ingestion")
	}
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

func (o *OrderBookStream) updateAndVerify(graph orderbook.OBGraph, sequence uint32) {
	status, err := o.getIngestionStatus()
	if err != nil {
		log.WithError(err).WithField("sequence", sequence).Info("Error obtaining ingestion status")
		return
	}

	dbUpdates, dbRemoved, err := o.update(status)
	if err != nil {
		log.WithError(err).WithField("sequence", sequence).Info("Error consuming from order book stream")
		return
	}
	ingestionUpdates, ingestionRemoved := graph.Pending()
	verifyUpdatedOffers(sequence, dbUpdates, ingestionUpdates)
	verifyRemovedOffers(sequence, dbRemoved, ingestionRemoved)
}

// Update will query the Horizon DB for offers which have been created, removed, or updated since the
// last time Update() was called. Those changes will then be applied to the in memory order book graph.
// After calling this function, the the in memory order book graph should be consistent with the
// Horizon DB (assuming no error is returned).
func (o *OrderBookStream) Update() error {
	if err := o.HistoryQ.BeginTx(&sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead}); err != nil {
		return errors.Wrap(err, "Error starting repeatable read transaction")
	}
	defer o.HistoryQ.Rollback()

	status, err := o.getIngestionStatus()
	if err != nil {
		return errors.Wrap(err, "Error obtaining ingestion status")
	}

	_, _, err = o.update(status)
	return err
}
