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

	accountID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	lastModifiedLedgerSeq := xdr.Uint32(123)
	asset := xdr.MustNewCreditAsset("USD", accountID)
	balanceID := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3},
	}
	cBalance := xdr.ClaimableBalanceEntry{
		BalanceId: balanceID,
		Claimants: []xdr.Claimant{
			xdr.Claimant{
				Type: xdr.ClaimantTypeClaimantTypeV0,
				V0: &xdr.ClaimantV0{
					Destination: xdr.MustAddress(accountID),
					Predicate: xdr.ClaimPredicate{
						Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
					},
				},
			},
		},
		Asset:  asset,
		Amount: 10,
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

	r, err := q.FindClaimableBalancesByDestination(xdr.MustAddress(accountID))
	tt.Assert.NoError(err)
	tt.Assert.Len(r, 1)

	removed, err := q.RemoveClaimableBalance(cBalance)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), removed)

	cbs := []ClaimableBalance{}
	err = q.Select(&cbs, selectClaimableBalances)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(cbs, 0)
	}
}

func TestFindClaimableBalancesByDestination(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	dest1 := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	dest2 := "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"

	lastModifiedLedgerSeq := xdr.Uint32(123)
	asset := xdr.MustNewCreditAsset("USD", dest1)
	balanceID := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3},
	}
	cBalance := xdr.ClaimableBalanceEntry{
		BalanceId: balanceID,
		Claimants: []xdr.Claimant{
			xdr.Claimant{
				Type: xdr.ClaimantTypeClaimantTypeV0,
				V0: &xdr.ClaimantV0{
					Destination: xdr.MustAddress(dest1),
					Predicate: xdr.ClaimPredicate{
						Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
					},
				},
			},
		},
		Asset:  asset,
		Amount: 10,
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

	balanceID = xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{3, 2, 1},
	}
	cBalance = xdr.ClaimableBalanceEntry{
		BalanceId: balanceID,
		Claimants: []xdr.Claimant{
			xdr.Claimant{
				Type: xdr.ClaimantTypeClaimantTypeV0,
				V0: &xdr.ClaimantV0{
					Destination: xdr.MustAddress(dest1),
					Predicate: xdr.ClaimPredicate{
						Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
					},
				},
			},
			xdr.Claimant{
				Type: xdr.ClaimantTypeClaimantTypeV0,
				V0: &xdr.ClaimantV0{
					Destination: xdr.MustAddress(dest2),
					Predicate: xdr.ClaimPredicate{
						Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
					},
				},
			},
		},
		Asset:  asset,
		Amount: 10,
	}
	entry = xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:             xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &cBalance,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
	}

	err = builder.Add(&entry)
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)

	cbs, err := q.FindClaimableBalancesByDestination(xdr.MustAddress(dest1))
	tt.Assert.NoError(err)
	tt.Assert.Len(cbs, 2)

	for _, cb := range cbs {
		tt.Assert.Equal(dest1, cb.Claimants[0].Destination)
	}

	cbs, err = q.FindClaimableBalancesByDestination(xdr.MustAddress(dest2))
	tt.Assert.NoError(err)
	tt.Assert.Len(cbs, 1)
	tt.Assert.Equal(dest2, cbs[0].Claimants[1].Destination)
}
