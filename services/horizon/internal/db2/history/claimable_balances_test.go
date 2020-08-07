package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestRemoveClaimableBalance(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	lastModifiedLedgerSeq := xdr.Uint32(123)
	asset := xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML")
	balanceID := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3},
	}
	cBalance := xdr.ClaimableBalanceEntry{
		BalanceId: balanceID,
		Claimants: []xdr.Claimant{},
		Asset:     asset,
		Amount:    10,
	}
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:             xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &cBalance,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
	}

	builder := q.NewClaimableBalancesBatchInsertBuilder(2)

	err := builder.Add(&entry)
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)

	removed, err := q.RemoveClaimableBalance(cBalance)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), removed)

	cbs := []ClaimableBalance{}
	err = q.Select(&cbs, selectClaimableBalances)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(cbs, 0)
	}
}
