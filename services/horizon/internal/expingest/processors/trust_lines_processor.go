package processors

import (
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type TrustLinesProcessor struct {
	TrustLinesQ history.QTrustLines

	cache                 *io.LedgerEntryChangeCache
	batchUpsertTrustLines []xdr.LedgerEntry
}

func (p *TrustLinesProcessor) Init(header xdr.LedgerHeader) error {
	p.init()
	return nil
}

func (p *TrustLinesProcessor) init() {
	p.batchUpsertTrustLines = []xdr.LedgerEntry{}
	p.cache = io.NewLedgerEntryChangeCache()
}

func (p *TrustLinesProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeTrustline {
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
		p.init()
	}

	return nil
}

func (p *TrustLinesProcessor) Commit() error {
	changes := p.cache.GetChanges()
	for _, change := range changes {
		var rowsAffected int64
		var err error
		var action string
		var ledgerKey xdr.LedgerKey

		switch {
		case change.Post != nil:
			// Created and updated
			p.batchUpsertTrustLines = append(p.batchUpsertTrustLines, *change.Post)
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			trustLine := change.Pre.Data.MustTrustLine()
			err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.TrustLinesQ.RemoveTrustLine(*ledgerKey.TrustLine)
			if err != nil {
				return err
			}

			if rowsAffected != 1 {
				return ingesterrors.NewStateError(errors.Errorf(
					"%d rows affected when %s trustline: %s %s",
					rowsAffected,
					action,
					ledgerKey.TrustLine.AccountId.Address(),
					ledgerKey.TrustLine.Asset.String(),
				))
			}
		default:
			return errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
		}
	}

	// Upsert accounts
	if len(p.batchUpsertTrustLines) > 0 {
		err := p.TrustLinesQ.UpsertTrustLines(p.batchUpsertTrustLines)
		if err != nil {
			return errors.Wrap(err, "errors in UpsertTrustLines")
		}
	}

	return nil
}
