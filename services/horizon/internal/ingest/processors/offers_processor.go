package processors

import (
	"context"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/offers"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
)

// The offers processor can be configured to trim the offers table
// by removing all offer rows which were marked for deletion at least 100 ledgers ago
const compactionWindow = uint32(100)

type OffersProcessor struct {
	offersQ  history.QOffers
	sequence uint32

	batchUpdateOffers  []history.Offer
	insertBatchBuilder history.OffersBatchInsertBuilder
}

func NewOffersProcessor(offersQ history.QOffers, sequence uint32) *OffersProcessor {
	p := &OffersProcessor{offersQ: offersQ, sequence: sequence}
	p.reset()
	return p
}

func (p *OffersProcessor) Name() string {
	return "processors.OffersProcessor"
}

func (p *OffersProcessor) reset() {
	p.batchUpdateOffers = []history.Offer{}
	p.insertBatchBuilder = p.offersQ.NewOffersBatchInsertBuilder()
}

func (p *OffersProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	event := offers.ProcessOffer(change)
	if event == nil {
		return nil
	}

	switch ev := event.(type) {
	case offers.OfferCreatedEvent:
		row := p.offerEventToRow(ev.OfferEventData)
		err := p.insertBatchBuilder.Add(row)
		if err != nil {
			return errors.New("Error adding to OffersBatchInsertBuilder")
		}
	case offers.OfferFillEvent:
		row := p.offerEventToRow(ev.OfferEventData)
		p.batchUpdateOffers = append(p.batchUpdateOffers, row)
	case offers.OfferClosedEvent:
		row := p.offerEventToRow(ev.OfferEventData)
		row.Deleted = true
		row.LastModifiedLedger = p.sequence
		p.batchUpdateOffers = append(p.batchUpdateOffers, row)
	default:
		return errors.New("Unknown offer event")

	}
	if p.insertBatchBuilder.Len()+len(p.batchUpdateOffers) > maxBatchSize {
		if err := p.flushCache(ctx); err != nil {
			return errors.Wrap(err, "error in Commit")
		}
	}

	return nil

}

func (p *OffersProcessor) offerEventToRow(e offers.OfferEventData) history.Offer {
	return history.Offer{
		SellerID:           e.SellerId,
		OfferID:            e.OfferID,
		SellingAsset:       e.SellingAsset,
		BuyingAsset:        e.BuyingAsset,
		Amount:             e.RemainingAmount,
		Pricen:             e.PriceN,
		Priced:             e.PriceD,
		Price:              float64(e.PriceN) / float64(e.PriceD),
		Flags:              e.Flags,
		LastModifiedLedger: e.LastModifiedLedger,
		Sponsor:            e.Sponsor,
	}
}

func (p *OffersProcessor) flushCache(ctx context.Context) error {
	defer p.reset()

	err := p.insertBatchBuilder.Exec(ctx)
	if err != nil {
		return errors.New("Error executing OffersBatchInsertBuilder")
	}

	if len(p.batchUpdateOffers) > 0 {
		err := p.offersQ.UpsertOffers(ctx, p.batchUpdateOffers)
		if err != nil {
			return errors.Wrap(err, "errors in UpsertOffers")
		}
	}

	return nil
}

func (p *OffersProcessor) Commit(ctx context.Context) error {
	if err := p.flushCache(ctx); err != nil {
		return errors.Wrap(err, "error flushing cache")
	}

	if p.sequence > compactionWindow {
		// trim offers table by removing offers which were deleted before the cutoff ledger
		if offerRowsRemoved, err := p.offersQ.CompactOffers(ctx, p.sequence-compactionWindow); err != nil {
			return errors.Wrap(err, "could not compact offers")
		} else {
			log.WithField("offer_rows_removed", offerRowsRemoved).Info("Trimmed offers table")
		}
	}

	return nil
}
