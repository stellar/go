package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type LiquidityPoolsChangeProcessor struct {
	qLiquidityPools history.QLiquidityPools
	cache           *ingest.ChangeCompactor
}

func NewLiquidityPoolsChangeProcessor(Q history.QLiquidityPools) *LiquidityPoolsChangeProcessor {
	p := &LiquidityPoolsChangeProcessor{
		qLiquidityPools: Q,
	}
	p.reset()
	return p
}

func (p *LiquidityPoolsChangeProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
}

func (p *LiquidityPoolsChangeProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeLiquidityPool {
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

func (p *LiquidityPoolsChangeProcessor) Commit(ctx context.Context) error {
	batch := p.qLiquidityPools.NewLiquidityPoolsBatchInsertBuilder(maxBatchSize)

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
			err = batch.Add(ctx, change.Post)
			rowsAffected = 1
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			lPool := change.Pre.Data.MustLiquidityPool()
			err = ledgerKey.SetLiquidityPool(lPool.LiquidityPoolId)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.qLiquidityPools.RemoveLiquidityPool(ctx, lPool)
		default:
			// Updated
			action = "updating"
			cBalance := change.Post.Data.MustLiquidityPool()
			err = ledgerKey.SetLiquidityPool(cBalance.LiquidityPoolId)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.qLiquidityPools.UpdateLiquidityPool(ctx, *change.Post)
		}

		if err != nil {
			return err
		}

		if rowsAffected != 1 {
			ledgerKeyString, err := ledgerKey.MarshalBinaryBase64()
			if err != nil {
				return errors.Wrap(err, "Error marshalling ledger key")
			}
			return ingest.NewStateError(errors.Errorf(
				"%d rows affected when %s liquidity pool: %s",
				rowsAffected,
				action,
				ledgerKeyString,
			))
		}
	}

	err := batch.Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "error executing batch")
	}

	return nil
}
