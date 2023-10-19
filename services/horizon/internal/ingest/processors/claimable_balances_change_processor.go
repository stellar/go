package processors

import (
	"context"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ClaimableBalancesChangeProcessor struct {
	encodingBuffer                *xdr.EncodingBuffer
	qClaimableBalances            history.QClaimableBalances
	cache                         *ingest.ChangeCompactor
	claimantsInsertBuilder        history.ClaimableBalanceClaimantBatchInsertBuilder
	claimableBalanceInsertBuilder history.ClaimableBalanceBatchInsertBuilder
	session                       db.SessionInterface
}

func NewClaimableBalancesChangeProcessor(Q history.QClaimableBalances, session db.SessionInterface) *ClaimableBalancesChangeProcessor {
	p := &ClaimableBalancesChangeProcessor{
		encodingBuffer:     xdr.NewEncodingBuffer(),
		qClaimableBalances: Q,
		session:            session,
	}
	p.reset()
	return p
}

func (p *ClaimableBalancesChangeProcessor) reset() {
	p.cache = ingest.NewChangeCompactor()
	p.claimantsInsertBuilder = p.qClaimableBalances.NewClaimableBalanceClaimantBatchInsertBuilder()
	p.claimableBalanceInsertBuilder = p.qClaimableBalances.NewClaimableBalanceBatchInsertBuilder()
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
		cbsToInsert   []history.ClaimableBalance
		cbIDsToDelete []string
	)
	changes := p.cache.GetChanges()
	for _, change := range changes {
		if change.Post != nil {
			// Created
			row, err := p.ledgerEntryToRow(change.Post)
			if err != nil {
				return err
			}
			cbsToInsert = append(cbsToInsert, row)
		} else {
			// Removed
			cBalance := change.Pre.Data.MustClaimableBalance()
			id, err := p.encodingBuffer.MarshalHex(cBalance.BalanceId)
			if err != nil {
				return err
			}
			cbIDsToDelete = append(cbIDsToDelete, id)
		}
	}

	if err := p.InsertClaimableBalanceAndClaimants(ctx, cbsToInsert); err != nil {
		return errors.Wrap(err, "error inserting claimable balance")
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

		// Remove ClaimableBalanceClaimants
		_, err = p.qClaimableBalances.RemoveClaimableBalanceClaimants(ctx, cbIDsToDelete)
		if err != nil {
			return errors.Wrap(err, "error executing removal of claimants")
		}
	}

	return nil
}

func (p *ClaimableBalancesChangeProcessor) InsertClaimableBalanceAndClaimants(ctx context.Context, claimableBalances []history.ClaimableBalance) error {
	if len(claimableBalances) == 0 {
		return nil
	}

	defer p.claimantsInsertBuilder.Reset()
	defer p.claimableBalanceInsertBuilder.Reset()

	for _, cb := range claimableBalances {

		if err := p.claimableBalanceInsertBuilder.Add(cb); err != nil {
			return errors.Wrap(err, "error executing insert")
		}
		// Add claimants
		for _, claimant := range cb.Claimants {
			claimant := history.ClaimableBalanceClaimant{
				BalanceID:          cb.BalanceID,
				Destination:        claimant.Destination,
				LastModifiedLedger: cb.LastModifiedLedger,
			}

			if err := p.claimantsInsertBuilder.Add(claimant); err != nil {
				return errors.Wrap(err, "error adding to claimantsInsertBuilder")
			}
		}
	}

	err := p.claimantsInsertBuilder.Exec(ctx, p.session)
	if err != nil {
		return errors.Wrap(err, "error executing claimableBalanceInsertBuilder")
	}

	err = p.claimableBalanceInsertBuilder.Exec(ctx, p.session)
	if err != nil {
		return errors.Wrap(err, "error executing claimantsInsertBuilder")
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

func (p *ClaimableBalancesChangeProcessor) ledgerEntryToRow(entry *xdr.LedgerEntry) (history.ClaimableBalance, error) {
	cBalance := entry.Data.MustClaimableBalance()
	id, err := xdr.MarshalHex(cBalance.BalanceId)
	if err != nil {
		return history.ClaimableBalance{}, err
	}
	row := history.ClaimableBalance{
		BalanceID:          id,
		Claimants:          buildClaimants(cBalance.Claimants),
		Asset:              cBalance.Asset,
		Amount:             cBalance.Amount,
		Sponsor:            ledgerEntrySponsorToNullString(*entry),
		LastModifiedLedger: uint32(entry.LastModifiedLedgerSeq),
		Flags:              uint32(cBalance.Flags()),
	}
	return row, nil
}
