package processors

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (p *DatabaseProcessor) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer r.Close()
	defer w.Close()

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
				err = p.HistoryQ.CreateAccountSigner(
					account,
					signer,
					weight,
				)
				if err != nil {
					return errors.Wrap(err, "Error updating account for signer")
				}
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

		switch p.Action {
		case AccountsForSigner:
			// TODO check if tx is success!
			err := p.processLedgerAccountsForSigner(transaction)
			if err != nil {
				return errors.Wrap(err, "Error in processLedgerAccountsForSigner")
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
				err := p.HistoryQ.RemoveAccountSigner(preAccountEntry.AccountId.Address(), signer)
				if err != nil {
					return errors.Wrap(err, "Error removing a signer")
				}
			}
		}

		if change.Post != nil {
			postAccountEntry := change.Post.MustAccount()
			for signer, weight := range postAccountEntry.SignerSummary() {
				err := p.HistoryQ.CreateAccountSigner(postAccountEntry.AccountId.Address(), signer, weight)
				if err != nil {
					return errors.Wrap(err, "Error inserting a signer")
				}
			}
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
