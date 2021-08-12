package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestFindLiquidityPool(t *testing.T) {
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

	lpObtained, err := q.FindLiquidityPoolByID(tt.Ctx, lp.PoolID)
	tt.Assert.NoError(err)

	tt.Assert.Equal(lp, lpObtained)
}

func TestRemoveLiquidityPool(t *testing.T) {
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

	lpObtained, err := q.FindLiquidityPoolByID(tt.Ctx, lp.PoolID)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(lpObtained)

	removed, err := q.RemoveLiquidityPool(tt.Ctx, lp.PoolID)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), removed)

	lps := []LiquidityPool{}
	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(lps, 0)
	}
}

func TestFindLiquidityPoolsByAssets(t *testing.T) {
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

	// query by no asset
	query := LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
	}

	lps, err := q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	// query by one asset
	query = LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
		Assets:    []xdr.Asset{xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML")},
	}

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	// query by two assets
	query = LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
		Assets: []xdr.Asset{
			xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			xdr.MustNewNativeAsset(),
		},
	}

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	// query by an asset not present
	query = LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
		Assets:    []xdr.Asset{xdr.MustNewCreditAsset("EUR", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML")},
	}

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 0)
}

func TestUpdateLiquidityPool(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	initialLP := LiquidityPool{
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

	err := builder.Add(tt.Ctx, initialLP)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)

	updatedLP := LiquidityPool{
		PoolID:         "cafebabedeadbeef000000000000000000000000000000000000000000000000",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            34,
		TrustlineCount: 52116,
		ShareCount:     100000,
		AssetReserves: []LiquidityPoolAssetReserve{
			{
				Asset:   xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				Reserve: 500,
			},
			{
				Asset:   xdr.MustNewNativeAsset(),
				Reserve: 600,
			},
		},
		LastModifiedLedger: 124,
	}

	updated, err := q.UpdateLiquidityPool(tt.Ctx, updatedLP)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), updated)

	lps := []LiquidityPool{}
	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)
	lp := lps[0]
	tt.Assert.Equal(updatedLP, lp)
}

func TestGetLiquidityPoolsByID(t *testing.T) {
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

	r, err := q.GetLiquidityPoolsByID(tt.Ctx, []string{lp.PoolID})
	tt.Assert.NoError(err)
	tt.Assert.Len(r, 1)

	removed, err := q.RemoveLiquidityPool(tt.Ctx, lp.PoolID)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), removed)

	r, err = q.GetLiquidityPoolsByID(tt.Ctx, []string{lp.PoolID})
	tt.Assert.NoError(err)
	tt.Assert.Len(r, 0)
}
