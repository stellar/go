package processors

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const maxBatchSize = 100000

func (p *DatabaseProcessor) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer r.Close()
	defer w.Close()

	var (
		accountSignerBatch history.AccountSignersBatchInsertBuilder
		offersBatch        history.OffersBatchInsertBuilder
	)

	switch p.Action {
	case AccountsForSigner:
		accountSignerBatch = p.SignersQ.NewAccountSignersBatchInsertBuilder(maxBatchSize)
	case Offers:
		offersBatch = p.OffersQ.NewOffersBatchInsertBuilder(maxBatchSize)
	default:
		return errors.Errorf("Invalid action type (%s)", p.Action)
	}

	for {
		entryChange, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if entryChange.Type != xdr.LedgerEntryChangeTypeLedgerEntryState {
			return errors.New("DatabaseProcessor requires LedgerEntryChangeTypeLedgerEntryState changes only")
		}

		switch p.Action {
		case AccountsForSigner:
			// We're interested in accounts only
			if entryChange.EntryType() != xdr.LedgerEntryTypeAccount {
				continue
			}

			accountEntry := entryChange.MustState().Data.MustAccount()
			account := accountEntry.AccountId.Address()
			for signer, weight := range accountEntry.SignerSummary() {
				err = accountSignerBatch.Add(history.AccountSigner{
					Account: account,
					Signer:  signer,
					Weight:  weight,
				})
				if err != nil {
					return errors.Wrap(err, "Error adding row to accountSignerBatch")
				}
			}
		case Offers:
			// We're interested in offers only
			if entryChange.EntryType() != xdr.LedgerEntryTypeOffer {
				continue
			}

			err = offersBatch.Add(
				entryChange.MustState().Data.MustOffer(),
				entryChange.MustState().LastModifiedLedgerSeq,
			)
			if err != nil {
				return errors.Wrap(err, "Error adding row to offersBatch")
			}
		default:
			return errors.New("Unknown action")
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	var err error

	switch p.Action {
	case AccountsForSigner:
		err = accountSignerBatch.Exec()
	case Offers:
		err = offersBatch.Exec()
	default:
		return errors.Errorf("Invalid action type (%s)", p.Action)
	}

	if err != nil {
		return errors.Wrap(err, "Error batch inserting rows")
	}

	return nil
}

func (p *DatabaseProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) error {
	defer r.Close()
	defer w.Close()

	for {
		transaction, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if transaction.Result.Result.Result.Code != xdr.TransactionResultCodeTxSuccess {
			continue
		}

		switch p.Action {
		case AccountsForSigner:
			err := p.processLedgerAccountsForSigner(transaction)
			if err != nil {
				return errors.Wrap(err, "Error in processLedgerAccountsForSigner")
			}
		case Offers:
			err := p.processLedgerOffers(transaction, r.GetSequence())
			if err != nil {
				return errors.Wrap(err, "Error in processLedgerOffers")
			}
		default:
			return errors.New("Unknown action")
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	return nil
}

func (p *DatabaseProcessor) processLedgerAccountsForSigner(transaction io.LedgerTransaction) error {
	for _, change := range transaction.GetChanges() {
		if change.Type != xdr.LedgerEntryTypeAccount {
			continue
		}

		if !change.AccountSignersChanged() {
			continue
		}

		// The code below removes all Pre signers adds Post signers but
		// can be improved by finding a diff (check performance first).
		if change.Pre != nil {
			preAccountEntry := change.Pre.MustAccount()
			for signer := range preAccountEntry.SignerSummary() {
				err := p.SignersQ.RemoveAccountSigner(preAccountEntry.AccountId.Address(), signer)
				if err != nil {
					return errors.Wrap(err, "Error removing a signer")
				}
			}
		}

		if change.Post != nil {
			postAccountEntry := change.Post.MustAccount()
			for signer, weight := range postAccountEntry.SignerSummary() {
				err := p.SignersQ.CreateAccountSigner(postAccountEntry.AccountId.Address(), signer, weight)
				if err != nil {
					return errors.Wrap(err, "Error inserting a signer")
				}
			}
		}
	}
	return nil
}

func (p *DatabaseProcessor) processLedgerOffers(transaction io.LedgerTransaction, currentLedger uint32) error {
	for _, change := range transaction.GetChanges() {
		if change.Type != xdr.LedgerEntryTypeOffer {
			continue
		}

		switch {
		case change.Post != nil:
			// Created or updated
			offer := change.Post.MustOffer()
			p.OffersQ.UpsertOffer(offer, xdr.Uint32(currentLedger))
		case change.Pre != nil && change.Post == nil:
			// Removed
			offer := change.Pre.MustOffer()
			p.OffersQ.RemoveOffer(offer.OfferId)
		}

	}
	return nil
}

func (p *DatabaseProcessor) Name() string {
	return fmt.Sprintf("DatabaseProcessor (%s)", p.Action)
}

func (p *DatabaseProcessor) Reset() {}

var _ ingestpipeline.StateProcessor = &DatabaseProcessor{}
var _ ingestpipeline.LedgerProcessor = &DatabaseProcessor{}
