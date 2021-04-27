package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type AccountDataProcessor struct {
	dataQ history.QData

	cache *ingest.ChangeCompactor
}

func NewAccountDataProcessor(dataQ history.QData) *AccountDataProcessor {
	p := &AccountDataProcessor{dataQ: dataQ}
	p.reset()
	return p
}

func (p *AccountDataProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
}

func (p *AccountDataProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	// We're interested in data only
	if change.Type != xdr.LedgerEntryTypeData {
		return nil
	}

	err := p.cache.AddChange(change)
	if err != nil {
		return errors.Wrap(err, "error adding to ledgerCache")
	}

	if p.cache.Size() > maxBatchSize {
		err = p.Commit(ctx)
		if err != nil {
			return errors.Wrap(err, "error in Commit")
		}
		p.reset()
	}

	return nil
}

func (p *AccountDataProcessor) Commit(ctx context.Context) error {
	batch := p.dataQ.NewAccountDataBatchInsertBuilder(maxBatchSize)

	changes := p.cache.GetChanges()
	for _, change := range changes {
		var err error
		var rowsAffected int64
		var action string
		var ledgerKey xdr.LedgerKey

		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			action = "inserting"
			err = batch.Add(ctx, *change.Post)
			rowsAffected = 1 // We don't track this when batch inserting
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			data := change.Pre.Data.MustData()
			err = ledgerKey.SetData(data.AccountId, string(data.DataName))
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.dataQ.RemoveAccountData(ctx, *ledgerKey.Data)
		default:
			// Updated
			action = "updating"
			data := change.Post.Data.MustData()
			err = ledgerKey.SetData(data.AccountId, string(data.DataName))
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.dataQ.UpdateAccountData(ctx, *change.Post)
		}

		if err != nil {
			return err
		}

		if rowsAffected != 1 {
			return ingest.NewStateError(errors.Errorf(
				"%d rows affected when %s data: %s %s",
				rowsAffected,
				action,
				ledgerKey.Data.AccountId.Address(),
				ledgerKey.Data.DataName,
			))
		}
	}

	err := batch.Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "error executing batch")
	}

	return nil
}
