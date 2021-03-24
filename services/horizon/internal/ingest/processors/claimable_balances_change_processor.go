package processors

import (
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ClaimableBalancesChangeProcessor struct {
	qClaimableBalances history.QClaimableBalances
	cache              *ingest.ChangeCompactor
}

func NewClaimableBalancesChangeProcessor(Q history.QClaimableBalances) *ClaimableBalancesChangeProcessor {
	p := &ClaimableBalancesChangeProcessor{
		qClaimableBalances: Q,
	}
	p.reset()
	return p
}

func (p *ClaimableBalancesChangeProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
}

func (p *ClaimableBalancesChangeProcessor) ProcessChange(change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeClaimableBalance {
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

func (p *ClaimableBalancesChangeProcessor) Commit() error {
	batch := p.qClaimableBalances.NewClaimableBalancesBatchInsertBuilder(maxBatchSize)

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
			err = batch.Add(change.Post)
			rowsAffected = 1
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			cBalance := change.Pre.Data.MustClaimableBalance()
			err = ledgerKey.SetClaimableBalance(cBalance.BalanceId)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.qClaimableBalances.RemoveClaimableBalance(cBalance)
		default:
			// Updated
			action = "updating"
			cBalance := change.Post.Data.MustClaimableBalance()
			err = ledgerKey.SetClaimableBalance(cBalance.BalanceId)
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.qClaimableBalances.UpdateClaimableBalance(*change.Post)
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
				"%d rows affected when %s claimable balance: %s",
				rowsAffected,
				action,
				ledgerKeyString,
			))
		}
	}

	err := batch.Exec()
	if err != nil {
		return errors.Wrap(err, "error executing batch")
	}

	return nil
}
