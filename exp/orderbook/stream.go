//lint:file-ignore U1000 this package is currently unused but it will be used in a future PR
package orderbook

import (
	"io"

	ingest "github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

func updateOrderbookWithStateStream(graph *OrderBookGraph, reader ingest.StateReadCloser) error {
	i := 0
	batch := graph.Batch()
	for {
		i++
		entry, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return errors.Wrap(err, "Error reading from StateReadCloser")
			}
		}

		if i%1000 == 0 {
			log.WithField("numEntries", i).Info("processed")
		}

		state, ok := entry.GetState()
		if !ok {
			continue
		}
		offer, ok := state.Data.GetOffer()
		if !ok {
			continue
		}
		log.WithField("offer", offer).Info("adding offer to graph")
		if offer.Price.N == 0 {
			log.WithField("offer", offer).Warn("offer has 0 price")
		} else {
			batch.AddOffer(offer)
		}
	}

	if err := batch.Apply(); err != nil {
		return errors.Wrap(err, "could not apply updates from StateReadCloser to graph")
	}

	return nil
}

func updateOrderbookWithLedgerEntryChanges(batch BatchedUpdates, changes xdr.LedgerEntryChanges) {
	for _, change := range changes {
		var offer xdr.OfferEntry
		var ok bool
		switch change.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			offer, ok = change.Created.Data.GetOffer()
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			offerKey, ok := change.Removed.GetOffer()
			if !ok {
				continue
			}
			batch.RemoveOffer(offerKey.OfferId)
			continue
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			offer, ok = change.State.Data.GetOffer()
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			offer, ok = change.Updated.Data.GetOffer()
		}
		if !ok {
			continue
		}

		batch.AddOffer(offer)
	}
}

func updateOrderbookWithLedgerStream(graph *OrderBookGraph, reader ingest.LedgerReadCloser) error {
	batch := graph.Batch()
	for {
		entry, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return errors.Wrap(err, "Error reading from LedgerReadCloser")
			}
		}

		if entry.Result.Result.Result.Code != xdr.TransactionResultCodeTxSuccess {
			continue
		}

		updateOrderbookWithLedgerEntryChanges(batch, entry.FeeChanges)

		if v1Meta, ok := entry.Meta.GetV1(); ok {
			updateOrderbookWithLedgerEntryChanges(batch, v1Meta.TxChanges)
			for _, operation := range v1Meta.Operations {
				updateOrderbookWithLedgerEntryChanges(batch, operation.Changes)
			}
		}
	}

	if err := batch.Apply(); err != nil {
		return errors.Wrap(err, "could not apply updates from LedgerReadCloser to graph")
	}

	return nil
}
