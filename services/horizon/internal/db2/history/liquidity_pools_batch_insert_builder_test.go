package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestAddLiquidityPool(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	lp := LiquidityPool{
		PoolID:         "cafebabedeadbeef000000000000000000000000000000000000000000000000",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            34,
		TrustlineCount: 52115,
		ShareCount:     412241,
		AssetReserves: []LiquidityPoolAssetReserve{
			{
				Asset:   xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				Reserve: 450,
			},
			{
				Asset:   xdr.MustNewNativeAsset(),
				Reserve: 450,
			},
		},
		LastModifiedLedger: 123,
	}

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, lp)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)
	tt.Assert.NoError(err)

	lps := []LiquidityPool{}
	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(lps, 1)
		lpObtained := lps[0]
		tt.Assert.Equal(lp, lpObtained)
	}
}
