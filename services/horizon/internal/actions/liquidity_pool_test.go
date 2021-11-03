package actions

import (
	"net/http/httptest"
	"testing"

	"github.com/guregu/null"
	"github.com/stellar/go/keypair"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

var (
	usdAsset = xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML")
	eurAsset = xdr.MustNewCreditAsset("EUR", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML")
)

func TestGetLiquidityPoolByID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}

	lp := history.LiquidityPool{
		PoolID:         "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            30,
		TrustlineCount: 100,
		ShareCount:     2000000000,
		AssetReserves: history.LiquidityPoolAssetReserves{
			{
				Asset:   xdr.MustNewNativeAsset(),
				Reserve: 100,
			},
			{
				Asset:   xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				Reserve: 200,
			},
		},
		LastModifiedLedger: 100,
	}

	err := q.UpsertLiquidityPools(tt.Ctx, []history.LiquidityPool{lp})
	tt.Assert.NoError(err)

	handler := GetLiquidityPoolByIDHandler{}
	response, err := handler.GetResource(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{},
		map[string]string{"liquidity_pool_id": lp.PoolID},
		q,
	))
	tt.Assert.NoError(err)

	resource := response.(protocol.LiquidityPool)
	tt.Assert.Equal(lp.PoolID, resource.ID)
	tt.Assert.Equal("constant_product", resource.Type)
	tt.Assert.Equal(uint32(30), resource.FeeBP)
	tt.Assert.Equal(uint64(100), resource.TotalTrustlines)
	tt.Assert.Equal("200.0000000", resource.TotalShares)

	tt.Assert.Equal("native", resource.Reserves[0].Asset)
	tt.Assert.Equal("0.0000100", resource.Reserves[0].Amount)

	tt.Assert.Equal("USD:GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Reserves[1].Asset)
	tt.Assert.Equal("0.0000200", resource.Reserves[1].Amount)

	// try to fetch pool which does not exist
	_, err = handler.GetResource(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{},
		map[string]string{"liquidity_pool_id": "123816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		q,
	))
	tt.Assert.Error(err)
	tt.Assert.True(q.NoRows(errors.Cause(err)))

	// try to fetch a random invalid hex id
	_, err = handler.GetResource(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{},
		map[string]string{"liquidity_pool_id": "0000001112122"},
		q,
	))
	tt.Assert.Error(err)
	p := err.(*problem.P)
	tt.Assert.Equal("bad_request", p.Type)
	tt.Assert.Equal("liquidity_pool_id", p.Extras["invalid_field"])
	tt.Assert.Equal("0000001112122 does not validate as sha256", p.Extras["reason"])
}

func TestGetLiquidityPools(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}

	lp1 := history.LiquidityPool{
		PoolID:         "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            30,
		TrustlineCount: 100,
		ShareCount:     2000000000,
		AssetReserves: history.LiquidityPoolAssetReserves{
			{
				Asset:   xdr.MustNewNativeAsset(),
				Reserve: 100,
			},
			{
				Asset:   xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				Reserve: 200,
			},
		},
		LastModifiedLedger: 100,
	}
	lp2 := history.LiquidityPool{
		PoolID:         "d827bf10a721d217de3cd9ab3f10198a54de558c093a511ec426028618df2633",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            30,
		TrustlineCount: 300,
		ShareCount:     4000000000,
		AssetReserves: history.LiquidityPoolAssetReserves{
			{
				Asset:   xdr.MustNewCreditAsset("EUR", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				Reserve: 300,
			},
			{
				Asset:   xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				Reserve: 400,
			},
		},
		LastModifiedLedger: 100,
	}
	err := q.UpsertLiquidityPools(tt.Ctx, []history.LiquidityPool{lp1, lp2})
	tt.Assert.NoError(err)

	handler := GetLiquidityPoolsHandler{}
	response, err := handler.GetResourcePage(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{},
		map[string]string{},
		q,
	))
	tt.Assert.NoError(err)
	tt.Assert.Len(response, 2)

	resource := response[0].(protocol.LiquidityPool)
	tt.Assert.Equal("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", resource.ID)
	tt.Assert.Equal("constant_product", resource.Type)
	tt.Assert.Equal(uint32(30), resource.FeeBP)
	tt.Assert.Equal(uint64(100), resource.TotalTrustlines)
	tt.Assert.Equal("200.0000000", resource.TotalShares)

	tt.Assert.Equal("native", resource.Reserves[0].Asset)
	tt.Assert.Equal("0.0000100", resource.Reserves[0].Amount)

	tt.Assert.Equal("USD:GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Reserves[1].Asset)
	tt.Assert.Equal("0.0000200", resource.Reserves[1].Amount)

	resource = response[1].(protocol.LiquidityPool)
	tt.Assert.Equal("d827bf10a721d217de3cd9ab3f10198a54de558c093a511ec426028618df2633", resource.ID)
	tt.Assert.Equal("constant_product", resource.Type)
	tt.Assert.Equal(uint32(30), resource.FeeBP)
	tt.Assert.Equal(uint64(300), resource.TotalTrustlines)
	tt.Assert.Equal("400.0000000", resource.TotalShares)

	tt.Assert.Equal("EUR:GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Reserves[0].Asset)
	tt.Assert.Equal("0.0000300", resource.Reserves[0].Amount)

	tt.Assert.Equal("USD:GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Reserves[1].Asset)
	tt.Assert.Equal("0.0000400", resource.Reserves[1].Amount)

	response, err = handler.GetResourcePage(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{"reserves": "native"},
		map[string]string{},
		q,
	))
	tt.Assert.NoError(err)
	tt.Assert.Len(response, 1)

	response, err = handler.GetResourcePage(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{"cursor": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		map[string]string{},
		q,
	))
	tt.Assert.NoError(err)
	tt.Assert.Len(response, 1)
	resource = response[0].(protocol.LiquidityPool)
	tt.Assert.Equal("d827bf10a721d217de3cd9ab3f10198a54de558c093a511ec426028618df2633", resource.ID)
}

func TestGetLiquidityPoolsByAccount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}

	// setup: create pools, then trustlines to some of them

	lp1 := history.MakeTestPool(xdr.MustNewNativeAsset(), 100, usdAsset, 200)
	lp2 := history.MakeTestPool(eurAsset, 300, usdAsset, 400)
	err := q.UpsertLiquidityPools(tt.Ctx, []history.LiquidityPool{lp1, lp2})
	tt.Assert.NoError(err)

	accountId := keypair.MustRandom().Address()

	tl1 := MakeTestTrustline(accountId, xdr.Asset{}, lp1.PoolID)
	err = q.UpsertTrustLines(tt.Ctx, []history.TrustLine{tl1})
	tt.Assert.NoError(err)

	// now perform the query and check results

	handler := GetLiquidityPoolsHandler{}
	response, err := handler.GetResourcePage(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{"account": accountId},
		map[string]string{},
		q,
	))
	tt.Assert.NoError(err)
	tt.Assert.Len(response, 1)
}

func MakeTestPool(A, B xdr.Asset, a, b uint64) history.LiquidityPool {
	if !A.LessThan(B) {
		B, A = A, B
		b, a = a, b
	}

	poolId, _ := xdr.NewPoolId(A, B, xdr.LiquidityPoolFeeV18)
	hexPoolId, _ := xdr.MarshalHex(poolId)
	return LiquidityPool{
		PoolID:         hexPoolId,
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            xdr.LiquidityPoolFeeV18,
		TrustlineCount: 12345,
		ShareCount:     67890,
		AssetReserves: []LiquidityPoolAssetReserve{
			{Asset: A, Reserve: a},
			{Asset: B, Reserve: b},
		},
		LastModifiedLedger: 123,
	}
}

func MakeTestTrustline(account string, asset xdr.Asset, poolId string) TrustLine {
	if (asset == xdr.Asset{} && poolId == "") ||
		(asset != xdr.Asset{} && poolId != "") {
		panic("can't make trustline to both asset and pool share")
	}

	trustline := TrustLine{
		AccountID:          account,
		Balance:            1000,
		LedgerKey:          "irrelevant",
		LiquidityPoolID:    poolId,
		Flags:              0,
		LastModifiedLedger: 1234,
		Sponsor:            null.String{},
	}

	if poolId == "" {
		trustline.AssetType = asset.Type
		switch asset.Type {
		case xdr.AssetTypeAssetTypeNative:
			trustline.AssetCode = "native"

		case xdr.AssetTypeAssetTypeCreditAlphanum4:
			fallthrough
		case xdr.AssetTypeAssetTypeCreditAlphanum12:
			trustline.AssetCode = asset.GetCode()
			trustline.AssetIssuer = asset.GetIssuer()

		default:
			panic("invalid asset type")
		}

		trustline.Limit = trustline.Balance * 10
		trustline.BuyingLiabilities = 1
		trustline.SellingLiabilities = 2
	} else {
		trustline.AssetType = xdr.AssetTypeAssetTypePoolShare
	}

	return trustline
}
