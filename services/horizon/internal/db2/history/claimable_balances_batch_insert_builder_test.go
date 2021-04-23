package history

import (
	"testing"

	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestAddClaimableBalance(t *testing.T) {
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
			{
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
		Ext: xdr.ClaimableBalanceEntryExt{
			V: 1,
			V1: &xdr.ClaimableBalanceEntryExtensionV1{
				Flags: xdr.Uint32(xdr.ClaimableBalanceFlagsClaimableBalanceClawbackEnabledFlag),
			},
		},
	}
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:             xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &cBalance,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	builder := q.NewClaimableBalancesBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, &entry)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)
	tt.Assert.NoError(err)

	cbs := []ClaimableBalance{}
	err = q.Select(tt.Ctx, &cbs, selectClaimableBalances)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(cbs, 1)

		cb := cbs[0]
		tt.Assert.Equal(balanceID, cb.BalanceID)
		tt.Assert.Len(cb.Claimants, 1)
		tt.Assert.Equal(accountID, cb.Claimants[0].Destination)
		tt.Assert.Equal(cBalance.Claimants[0].MustV0().Predicate, cb.Claimants[0].Predicate)
		tt.Assert.Equal(asset, cb.Asset)
		tt.Assert.Equal(null.NewString("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", true), cb.Sponsor)
		tt.Assert.Equal(uint32(lastModifiedLedgerSeq), cb.LastModifiedLedger)
		tt.Assert.Equal(uint32(xdr.ClaimableBalanceFlagsClaimableBalanceClawbackEnabledFlag), cb.Flags)
	}
}
