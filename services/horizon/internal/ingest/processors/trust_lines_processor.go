package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type TrustLinesProcessor struct {
	trustLinesQ history.QTrustLines

	cache *ingest.ChangeCompactor
}

func NewTrustLinesProcessor(trustLinesQ history.QTrustLines) *TrustLinesProcessor {
	p := &TrustLinesProcessor{trustLinesQ: trustLinesQ}
	p.reset()
	return p
}

func (p *TrustLinesProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
}

func (p *TrustLinesProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeTrustline {
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

func (p *TrustLinesProcessor) Commit(ctx context.Context) error {
	batchUpsertTrustLines := []xdr.LedgerEntry{}

	changes := p.cache.GetChanges()
	for _, change := range changes {
		var rowsAffected int64
		var err error
		var action string
		var ledgerKey xdr.LedgerKey

		switch {
		case change.Post != nil:
			// Created and updated
			batchUpsertTrustLines = append(batchUpsertTrustLines, *change.Post)
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			trustLine := change.Pre.Data.MustTrustLine()
			err = ledgerKey.SetTrustline(trustLine.AccountId, trustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.trustLinesQ.RemoveTrustLine(ctx, *ledgerKey.TrustLine)
			if err != nil {
				return err
			}

			if rowsAffected != 1 {
				return ingest.NewStateError(errors.Errorf(
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
	if len(batchUpsertTrustLines) > 0 {
		err := p.trustLinesQ.UpsertTrustLines(ctx, batchUpsertTrustLines)
		if err != nil {
			return errors.Wrap(err, "errors in UpsertTrustLines")
		}
	}

	return nil
}
