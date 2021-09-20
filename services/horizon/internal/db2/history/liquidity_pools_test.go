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

	err := q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{lp})
	tt.Assert.NoError(err)

	lpObtained, err := q.FindLiquidityPoolByID(tt.Ctx, lp.PoolID)
	tt.Assert.NoError(err)

	tt.Assert.Equal(lp, lpObtained)
}

func removeLiquidityPool(t *test.T, q *Q, lp LiquidityPool, sequence uint32) {
	removed := lp
	removed.Deleted = true
	removed.LastModifiedLedger = sequence
	err := q.UpsertLiquidityPools(t.Ctx, []LiquidityPool{removed})
	t.Assert.NoError(err)
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

	err := q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{lp})
	tt.Assert.NoError(err)

	count, err := q.CountLiquidityPools(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Equal(1, count)

	lpObtained, err := q.FindLiquidityPoolByID(tt.Ctx, lp.PoolID)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(lpObtained)

	removeLiquidityPool(tt, q, lp, 200)

	_, err = q.FindLiquidityPoolByID(tt.Ctx, lp.PoolID)
	tt.Assert.EqualError(err, "sql: no rows in result set")

	count, err = q.CountLiquidityPools(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Equal(0, count)

	lps := []LiquidityPool{}
	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)
	tt.Assert.NoError(err)

	tt.Assert.Len(lps, 1)
	expected := lp
	expected.Deleted = true
	expected.LastModifiedLedger = 200
	tt.Assert.Equal(expected, lps[0])

	lp.LastModifiedLedger = 250
	lp.Deleted = false
	lp.ShareCount = 1
	lp.TrustlineCount = 2
	err = q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{lp})
	tt.Assert.NoError(err)
	tt.Assert.NoError(err)

	lpObtained, err = q.FindLiquidityPoolByID(tt.Ctx, lp.PoolID)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(lpObtained)

	tt.Assert.Equal(lp, lpObtained)
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

	err := q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{lp})
	tt.Assert.NoError(err)

	// query by no asset
	query := LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
	}

	lps, err := q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	pool := lps[0]
	lps, err = q.GetAllLiquidityPools(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)
	tt.Assert.Equal(pool, lps[0])

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

	removeLiquidityPool(tt, q, lp, 200)

	query = LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
	}

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 0)

	// query by one asset
	query = LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
		Assets:    []xdr.Asset{xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML")},
	}

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 0)
}

func TestLiquidityPoolCompaction(t *testing.T) {
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

	err := q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{lp})
	tt.Assert.NoError(err)

	compationSequence, err := q.GetLiquidityPoolCompactionSequence(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Equal(uint32(0), compationSequence)

	rowsRemoved, err := q.CompactLiquidityPools(tt.Ctx, lp.LastModifiedLedger)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), rowsRemoved)

	compationSequence, err = q.GetLiquidityPoolCompactionSequence(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Equal(lp.LastModifiedLedger, compationSequence)

	// query by no asset
	query := LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
	}

	lps, err := q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	removeLiquidityPool(tt, q, lp, 200)

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 0)

	lps, err = q.GetAllLiquidityPools(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 0)

	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	rowsRemoved, err = q.CompactLiquidityPools(tt.Ctx, 199)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), rowsRemoved)
	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	rowsRemoved, err = q.CompactLiquidityPools(tt.Ctx, 200)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsRemoved)
	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)
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

	err := q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{initialLP})
	tt.Assert.NoError(err)

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

	err = q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{updatedLP})
	tt.Assert.NoError(err)

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

	err := q.UpsertLiquidityPools(tt.Ctx, []LiquidityPool{lp})
	tt.Assert.NoError(err)

	r, err := q.GetLiquidityPoolsByID(tt.Ctx, []string{lp.PoolID})
	tt.Assert.NoError(err)
	tt.Assert.Len(r, 1)

	removeLiquidityPool(tt, q, lp, 200)

	r, err = q.GetLiquidityPoolsByID(tt.Ctx, []string{lp.PoolID})
	tt.Assert.NoError(err)
	tt.Assert.Len(r, 0)
}
