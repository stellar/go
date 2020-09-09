package resourceadapter

import (
	"encoding/json"
	"testing"

	"github.com/guregu/null"
	. "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestPopulateClaimableBalance(t *testing.T) {
	tt := assert.New(t)
	ctx, _ := test.ContextWithLogBuffer()
	resource := ClaimableBalance{}

	balanceID := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3},
	}
	claimableBalance := history.ClaimableBalance{
		BalanceID: balanceID,
		Asset:     xdr.MustNewNativeAsset(),
		Amount:    100000000,
		Sponsor:   null.StringFrom("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Claimants: history.Claimants{
			{
				Destination: "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
				Predicate: xdr.ClaimPredicate{
					Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
				},
			},
		},
		LastModifiedLedger: 123,
	}

	err := PopulateClaimableBalance(ctx, &resource, claimableBalance)
	tt.NoError(err)

	tt.Equal("000000000102030000000000000000000000000000000000000000000000000000000000", resource.BalanceID)
	tt.Equal(claimableBalance.Asset.StringCanonical(), resource.Asset)
	tt.Equal("10.0000000", resource.Amount)
	tt.Equal("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Sponsor)
	tt.Equal(uint32(123), resource.LastModifiedLedger)
	tt.Len(resource.Claimants, 1)
	tt.Equal("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Claimants[0].Destination)
	tt.Equal("123-000000000102030000000000000000000000000000000000000000000000000000000000", resource.PagingToken())

	links, err := json.Marshal(resource.Links)
	tt.NoError(err)
	want := `
	{
	  "self": {
		"href": "/claimable_balances/000000000102030000000000000000000000000000000000000000000000000000000000"
	  }
	}
	`
	tt.JSONEq(want, string(links))

	predicate, err := json.Marshal(resource.Claimants[0].Predicate)
	tt.NoError(err)
	tt.JSONEq(`{"unconditional":true}`, string(predicate))
}
