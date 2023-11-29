package processors

import (
	"context"

	"github.com/guregu/null"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type SignersProcessor struct {
	signersQ history.QSigners

	cache              *ingest.ChangeCompactor
	batchInsertBuilder history.AccountSignersBatchInsertBuilder
	// insertOnlyMode is a mode in which we don't use ledger cache and we just
	// add signers to a batch, then we Exec all signers in one insert query.
	// This is done to make history buckets processing faster (batch inserting).
	useLedgerEntryCache bool
}

func NewSignersProcessor(
	signersQ history.QSigners, useLedgerEntryCache bool,
) *SignersProcessor {
	p := &SignersProcessor{signersQ: signersQ, useLedgerEntryCache: useLedgerEntryCache}
	p.reset()
	return p
}

func (p *SignersProcessor) reset() {
	p.batchInsertBuilder = p.signersQ.NewAccountSignersBatchInsertBuilder()
	p.cache = ingest.NewChangeCompactor()
}

func (p *SignersProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeAccount {
		return nil
	}

	if p.useLedgerEntryCache {
		err := p.cache.AddChange(change)
		if err != nil {
			return errors.Wrap(err, "error adding to ledgerCache")
		}

		if p.cache.Size() > maxBatchSize {
			err = p.Commit(ctx)
			if err != nil {
				return errors.Wrap(err, "error in Commit")
			}
		}

		return nil
	}

	if change.Pre == nil && change.Post != nil {
		postAccountEntry := change.Post.Data.MustAccount()
		if err := p.addAccountSigners(postAccountEntry); err != nil {
			return err
		}
	} else {
		return errors.New("SignersProcessor is in insert only mode")
	}

	return nil
}

func (p *SignersProcessor) Commit(ctx context.Context) error {
	defer p.reset()

	if p.useLedgerEntryCache {
		changes := p.cache.GetChanges()
		for _, change := range changes {
			if !change.AccountSignersChanged() {
				continue
			}

			// The code below removes all Pre signers adds Post signers but
			// can be improved by finding a diff (check performance first).
			if change.Pre != nil {
				if err := p.removeAccountSigners(ctx, change.Pre.Data.MustAccount()); err != nil {
					return err
				}
			}

			if change.Post != nil {
				if err := p.addAccountSigners(change.Post.Data.MustAccount()); err != nil {
					return err
				}
			}
		}
	}

	err := p.batchInsertBuilder.Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "error executing AccountSignersBatchInsertBuilder")
	}

	return nil
}

func (p *SignersProcessor) removeAccountSigners(ctx context.Context, accountEntry xdr.AccountEntry) error {
	for signer := range accountEntry.SignerSummary() {
		rowsAffected, err := p.signersQ.RemoveAccountSigner(ctx, accountEntry.AccountId.Address(), signer)
		if err != nil {
			return errors.Wrap(err, "Error removing a signer")
		}

		if rowsAffected != 1 {
			return ingest.NewStateError(errors.Errorf(
				"Expected account=%s signer=%s in database but not found when removing (rows affected = %d)",
				accountEntry.AccountId.Address(),
				signer,
				rowsAffected,
			))
		}
	}
	return nil
}

func (p *SignersProcessor) addAccountSigners(accountEntry xdr.AccountEntry) error {
	sponsorsPerSigner := accountEntry.SponsorPerSigner()
	for signer, weight := range accountEntry.SignerSummary() {
		// Ignore master key
		var sponsor null.String
		if signer != accountEntry.AccountId.Address() {
			if sponsorDesc, isSponsored := sponsorsPerSigner[signer]; isSponsored {
				sponsor = null.StringFrom(sponsorDesc.Address())
			}
		}

		if err := p.batchInsertBuilder.Add(history.AccountSigner{
			Account: accountEntry.AccountId.Address(),
			Signer:  signer,
			Weight:  weight,
			Sponsor: sponsor,
		}); err != nil {
			return errors.Wrapf(err, "Error adding signer (%s) to AccountSignersBatchInsertBuilder", signer)
		}
	}
	return nil
}
