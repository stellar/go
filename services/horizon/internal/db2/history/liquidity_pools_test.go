package history

import (
	"testing"

	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestFindLiquidityPool(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	accountID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	lastModifiedLedgerSeq := xdr.Uint32(123)
	lPoolEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", accountID),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 123,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPoolEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, &entry)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)

	lp, err := q.FindLiquidityPoolByID(tt.Ctx, lPoolEntry.LiquidityPoolId)
	tt.Assert.NoError(err)

	cp := lPoolEntry.Body.ConstantProduct
	tt.Assert.Equal(lPoolEntry.LiquidityPoolId, lp.PoolID)
	tt.Assert.Equal(uint32(cp.Params.Fee), lp.Fee)
	tt.Assert.Equal(uint64(cp.TotalPoolShares), lp.ShareCount)
	tt.Assert.Equal(uint64(cp.PoolSharesTrustLineCount), lp.TrustlineCount)
	tt.Assert.Len(lp.AssetReserves, 2)
	tt.Assert.Equal(buildLiquidityPoolAssetReserves(*cp), lp.AssetReserves)
	tt.Assert.Equal(null.NewString("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", true), lp.Sponsor)
	tt.Assert.Equal(uint32(lastModifiedLedgerSeq), lp.LastModifiedLedger)
}

func TestRemoveLiquidityPool(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	accountID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	lastModifiedLedgerSeq := xdr.Uint32(123)
	lPoolEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", accountID),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 123,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPoolEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, &entry)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)

	lp, err := q.FindLiquidityPoolByID(tt.Ctx, lPoolEntry.LiquidityPoolId)
	tt.Assert.NoError(err)
	tt.Assert.NotNil(lp)

	removed, err := q.RemoveLiquidityPool(tt.Ctx, lPoolEntry)
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

	accountID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	lastModifiedLedgerSeq := xdr.Uint32(123)
	lPoolEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", accountID),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 123,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPoolEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, &entry)
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
		Assets:    []xdr.Asset{xdr.MustNewCreditAsset("USD", accountID)},
	}

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	// query by two assets
	query = LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
		Assets:    []xdr.Asset{xdr.MustNewCreditAsset("USD", accountID), xdr.MustNewNativeAsset()},
	}

	lps, err = q.GetLiquidityPools(tt.Ctx, query)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)

	// query by an asset not present
	query = LiquidityPoolsQuery{
		PageQuery: db2.MustPageQuery("", false, "", 10),
		Assets:    []xdr.Asset{xdr.MustNewCreditAsset("EUR", accountID)},
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

	accountID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	lastModifiedLedgerSeq := xdr.Uint32(123)
	initialPoolEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", accountID),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 123,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	initialEntry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &initialPoolEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
	}

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, &initialEntry)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)

	updatedLastModifiedLedgerSeq := xdr.Uint32(124)
	updatedPoolEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", accountID),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 500,
				ReserveB:                 600,
				TotalPoolShares:          100000,
				PoolSharesTrustLineCount: 52116,
			},
		},
	}
	updatedEntry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &updatedPoolEntry,
		},
		LastModifiedLedgerSeq: updatedLastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	updated, err := q.UpdateLiquidityPool(tt.Ctx, updatedEntry)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), updated)

	lps := []LiquidityPool{}
	err = q.Select(tt.Ctx, &lps, selectLiquidityPools)
	tt.Assert.NoError(err)
	tt.Assert.Len(lps, 1)
	lp := lps[0]
	cp := updatedPoolEntry.Body.ConstantProduct

	tt.Assert.Equal(updatedPoolEntry.LiquidityPoolId, lp.PoolID)
	tt.Assert.Equal(uint32(cp.Params.Fee), lp.Fee)
	tt.Assert.Equal(uint64(cp.TotalPoolShares), lp.ShareCount)
	tt.Assert.Equal(uint64(cp.PoolSharesTrustLineCount), lp.TrustlineCount)
	tt.Assert.Len(lp.AssetReserves, 2)
	tt.Assert.Equal(buildLiquidityPoolAssetReserves(*cp), lp.AssetReserves)
	tt.Assert.Equal(null.NewString("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", true), lp.Sponsor)
	tt.Assert.Equal(uint32(updatedLastModifiedLedgerSeq), lp.LastModifiedLedger)
}

func TestGetLiquidityPoolsByID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	accountID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	lastModifiedLedgerSeq := xdr.Uint32(123)
	lPoolEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", accountID),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 123,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPoolEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, &entry)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)

	r, err := q.GetLiquidityPoolsByID(tt.Ctx, []xdr.PoolId{lPoolEntry.LiquidityPoolId})
	tt.Assert.NoError(err)
	tt.Assert.Len(r, 1)

	removed, err := q.RemoveLiquidityPool(tt.Ctx, lPoolEntry)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), removed)

	r, err = q.GetLiquidityPoolsByID(tt.Ctx, []xdr.PoolId{lPoolEntry.LiquidityPoolId})
	tt.Assert.NoError(err)
	tt.Assert.Len(r, 0)
}
