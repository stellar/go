package processors

import (
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ChangesMode int

const (
	InitChangesMode ChangesMode = iota
	LedgerChangesMode
)

type AccountDataProcessor struct {
	DataQ history.QData

	mode  ChangesMode
	cache *io.LedgerEntryChangeCache
	batch history.AccountDataBatchInsertBuilder
}

func (p *AccountDataProcessor) Init(header xdr.LedgerHeader, mode ChangesMode) error {
	p.mode = mode
	p.batch = nil
	p.cache = nil
	switch p.mode {
	case InitChangesMode:
		p.batch = p.DataQ.NewAccountDataBatchInsertBuilder(maxBatchSize)
	case LedgerChangesMode:
		p.cache = io.NewLedgerEntryChangeCache()
	default:
		return errors.New("Invalid changes mode")
	}
	return nil
}

func (p *AccountDataProcessor) ProcessChange(change io.Change) error {
	// We're interested in data only
	if change.Type != xdr.LedgerEntryTypeData {
		return nil
	}

	switch p.mode {
	case InitChangesMode:
		err := p.batch.Add(
			change.Post.Data.MustData(),
			change.Post.LastModifiedLedgerSeq,
		)
		if err != nil {
			return errors.Wrap(err, "error adding row to batch")
		}
	case LedgerChangesMode:
		err := p.cache.AddChange(change)
		if err != nil {
			return errors.Wrap(err, "error adding to ledgerCache")
		}
	default:
		return errors.New("Invalid changes mode")
	}

	return nil
}

func (p *AccountDataProcessor) Commit() error {
	switch p.mode {
	case InitChangesMode:
		err := p.batch.Exec()
		if err != nil {
			return errors.Wrap(err, "error executing batch")
		}
	case LedgerChangesMode:
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
				data := change.Post.Data.MustData()
				err = ledgerKey.SetData(data.AccountId, string(data.DataName))
				if err != nil {
					return errors.Wrap(err, "Error creating ledger key")
				}
				rowsAffected, err = p.DataQ.InsertAccountData(data, change.Post.LastModifiedLedgerSeq)
			case change.Pre != nil && change.Post == nil:
				// Removed
				action = "removing"
				data := change.Pre.Data.MustData()
				err = ledgerKey.SetData(data.AccountId, string(data.DataName))
				if err != nil {
					return errors.Wrap(err, "Error creating ledger key")
				}
				rowsAffected, err = p.DataQ.RemoveAccountData(*ledgerKey.Data)
			default:
				// Updated
				action = "updating"
				data := change.Post.Data.MustData()
				err = ledgerKey.SetData(data.AccountId, string(data.DataName))
				if err != nil {
					return errors.Wrap(err, "Error creating ledger key")
				}
				rowsAffected, err = p.DataQ.UpdateAccountData(data, change.Post.LastModifiedLedgerSeq)
			}

			if err != nil {
				return err
			}

			if rowsAffected != 1 {
				return ingesterrors.NewStateError(errors.Errorf(
					"%d rows affected when %s data: %s %s",
					rowsAffected,
					action,
					ledgerKey.Data.AccountId.Address(),
					ledgerKey.Data.DataName,
				))
			}
		}
	default:
		return errors.New("Invalid changes mode")
	}
	return nil
}
