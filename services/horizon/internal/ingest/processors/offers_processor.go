package processors

import (
	ingesterrors "github.com/stellar/go/ingest/errors"
	"github.com/stellar/go/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// The offers processor can be configured to trim the offers table
// by removing all offer rows which were marked for deletion at least 100 ledgers ago
const offerCompactionWindow = uint32(100)

type OffersProcessor struct {
	offersQ  history.QOffers
	sequence uint32

	cache       *io.LedgerEntryChangeCache
	insertBatch history.OffersBatchInsertBuilder
	removeBatch []int64
}

func NewOffersProcessor(offersQ history.QOffers, sequence uint32) *OffersProcessor {
	p := &OffersProcessor{offersQ: offersQ, sequence: sequence}
	p.reset()
	return p
}

func (p *OffersProcessor) reset() {
	p.cache = io.NewLedgerEntryChangeCache()
	p.insertBatch = p.offersQ.NewOffersBatchInsertBuilder(maxBatchSize)
	p.removeBatch = []int64{}
}

func (p *OffersProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeOffer {
		return nil
	}

	if err := p.cache.AddChange(change); err != nil {
		return errors.Wrap(err, "error adding to ledgerCache")
	}

	if p.cache.Size() > maxBatchSize {
		if err := p.flushCache(); err != nil {
			return errors.Wrap(err, "error in Commit")
		}
		p.reset()
	}

	return nil
}

func (p *OffersProcessor) ledgerEntryToRow(entry *xdr.LedgerEntry) history.Offer {
	offer := entry.Data.MustOffer()
	return history.Offer{
		SellerID:           offer.SellerId.Address(),
		OfferID:            int64(offer.OfferId),
		SellingAsset:       offer.Selling,
		BuyingAsset:        offer.Buying,
		Amount:             int64(offer.Amount),
		Pricen:             int32(offer.Price.N),
		Priced:             int32(offer.Price.D),
		Price:              float64(offer.Price.N) / float64(offer.Price.D),
		Flags:              uint32(offer.Flags),
		LastModifiedLedger: uint32(entry.LastModifiedLedgerSeq),
		Sponsor:            ledgerEntrySponsorToNullString(*entry),
	}
}

func (p *OffersProcessor) flushCache() error {
	changes := p.cache.GetChanges()
	for _, change := range changes {
		var rowsAffected int64
		var err error
		var action string
		var offerID xdr.Int64

		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			action = "inserting"
			row := p.ledgerEntryToRow(change.Post)
			err = p.insertBatch.Add(row)
			rowsAffected = 1 // We don't track this when batch inserting
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			offer := change.Pre.Data.MustOffer()
			p.removeBatch = append(p.removeBatch, int64(offer.OfferId))
			rowsAffected = 1 // We don't track this when batch removing
		default:
			// Updated
			action = "updating"
			offer := change.Post.Data.MustOffer()
			offerID = offer.OfferId
			row := p.ledgerEntryToRow(change.Post)
			rowsAffected, err = p.offersQ.UpdateOffer(row)
		}

		if err != nil {
			return err
		}

		if rowsAffected != 1 {
			return ingesterrors.NewStateError(errors.Errorf(
				"%d rows affected when %s offer %d",
				rowsAffected,
				action,
				offerID,
			))
		}
	}

	err := p.insertBatch.Exec()
	if err != nil {
		return errors.Wrap(err, "error executing batch")
	}

	if len(p.removeBatch) > 0 {
		_, err = p.offersQ.RemoveOffers(p.removeBatch, p.sequence)
		if err != nil {
			return errors.Wrap(err, "error in RemoveOffers")
		}
	}

	return nil
}

func (p *OffersProcessor) Commit() error {
	if err := p.flushCache(); err != nil {
		return errors.Wrap(err, "error flushing cache")
	}

	if p.sequence > offerCompactionWindow {
		// trim offers table by removing offers which were deleted before the cutoff ledger
		if offerRowsRemoved, err := p.offersQ.CompactOffers(p.sequence - offerCompactionWindow); err != nil {
			return errors.Wrap(err, "could not compact offers")
		} else {
			log.WithField("offer_rows_removed", offerRowsRemoved).Info("Trimmed offers table")
		}
	}

	return nil
}
