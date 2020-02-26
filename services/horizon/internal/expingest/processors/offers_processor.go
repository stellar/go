package processors

import (
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type OffersProcessor struct {
	offersQ history.QOffers

	cache *io.LedgerEntryChangeCache
	batch history.OffersBatchInsertBuilder
}

func NewOffersProcessor(offersQ history.QOffers) *OffersProcessor {
	p := &OffersProcessor{offersQ: offersQ}
	p.reset()
	return p
}

func (p *OffersProcessor) reset() {
	p.batch = p.offersQ.NewOffersBatchInsertBuilder(maxBatchSize)
	p.cache = io.NewLedgerEntryChangeCache()
}

func (p *OffersProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeOffer {
		return nil
	}

	err := p.cache.AddChange(change)
	if err != nil {
		return errors.Wrap(err, "error adding to ledgerCache")
	}

	if p.cache.Size() > maxBatchSize {
		err = p.Commit()
		if err != nil {
			return errors.Wrap(err, "error in Commit")
		}
		p.reset()
	}

	return nil
}

func (p *OffersProcessor) Commit() error {
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
			err = p.batch.Add(
				change.Post.Data.MustOffer(),
				change.Post.LastModifiedLedgerSeq,
			)
			rowsAffected = 1 // We don't track this when batch inserting
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			offer := change.Pre.Data.MustOffer()
			offerID = offer.OfferId
			rowsAffected, err = p.offersQ.RemoveOffer(offer.OfferId)
		default:
			// Updated
			action = "updating"
			offer := change.Post.Data.MustOffer()
			offerID = offer.OfferId
			rowsAffected, err = p.offersQ.UpdateOffer(offer, change.Post.LastModifiedLedgerSeq)
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

	err := p.batch.Exec()
	if err != nil {
		return errors.Wrap(err, "error executing batch")
	}

	return nil
}
