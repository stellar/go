package processors

import (
	"context"

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

func (p *ClaimableBalancesChangeProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	if change.Type != xdr.LedgerEntryTypeClaimableBalance {
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

func (p *ClaimableBalancesChangeProcessor) Commit(ctx context.Context) error {
	var (
		cbsToUpsert   []history.ClaimableBalance
		cbIDsToDelete []xdr.ClaimableBalanceId
	)
	changes := p.cache.GetChanges()
	for _, change := range changes {
		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			cbsToUpsert = append(cbsToUpsert, p.ledgerEntryToRow(change.Post))
		case change.Pre != nil && change.Post == nil:
			// Removed
			cBalance := change.Pre.Data.MustClaimableBalance()
			cbIDsToDelete = append(cbIDsToDelete, cBalance.BalanceId)
		default:
			// Updated
			cbsToUpsert = append(cbsToUpsert, p.ledgerEntryToRow(change.Post))
		}
	}

	if len(cbsToUpsert) > 0 {
		if err := p.qClaimableBalances.UpsertClaimableBalances(ctx, cbsToUpsert); err != nil {
			return errors.Wrap(err, "error executing upsert")
		}
	}

	if len(cbIDsToDelete) > 0 {
		count, err := p.qClaimableBalances.RemoveClaimableBalances(ctx, cbIDsToDelete)
		if err != nil {
			return errors.Wrap(err, "error executing removal")
		}
		if count != int64(len(cbIDsToDelete)) {
			return ingest.NewStateError(errors.Errorf(
				"%d rows affected when deleting %d claimable balances",
				count,
				len(cbIDsToDelete),
			))
		}
	}

	return nil
}

func buildClaimants(claimants []xdr.Claimant) history.Claimants {
	hClaimants := history.Claimants{}
	for _, c := range claimants {
		xc := c.MustV0()
		hClaimants = append(hClaimants, history.Claimant{
			Destination: xc.Destination.Address(),
			Predicate:   xc.Predicate,
		})
	}
	return hClaimants
}

func (p *ClaimableBalancesChangeProcessor) ledgerEntryToRow(entry *xdr.LedgerEntry) history.ClaimableBalance {
	cBalance := entry.Data.MustClaimableBalance()
	return history.ClaimableBalance{
		BalanceID:          cBalance.BalanceId,
		Claimants:          buildClaimants(cBalance.Claimants),
		Asset:              cBalance.Asset,
		Amount:             cBalance.Amount,
		Sponsor:            ledgerEntrySponsorToNullString(*entry),
		LastModifiedLedger: uint32(entry.LastModifiedLedgerSeq),
		Flags:              uint32(cBalance.Flags()),
	}

}
