package actions

import (
	"net/http/httptest"
	"testing"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
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
		ShareCount:     200,
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

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)
	err := builder.Add(tt.Ctx, lp)
	tt.Assert.NoError(err)
	err = builder.Exec(tt.Ctx)
	tt.Assert.NoError(err)

	handler := GetLiquidityPoolByIDHandler{}
	response, err := handler.GetResource(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{},
		map[string]string{"id": lp.PoolID},
		q,
	))
	tt.Assert.NoError(err)

	resource := response.(protocol.LiquidityPool)
	tt.Assert.Equal(lp.PoolID, resource.ID)
	tt.Assert.Equal("constant_product", resource.Type)
	tt.Assert.Equal(uint32(30), resource.FeeBP)
	tt.Assert.Equal(uint64(100), resource.TotalTrustlines)
	tt.Assert.Equal(uint64(200), resource.TotalShares)

	tt.Assert.Equal("native", resource.Reserves[0].Asset)
	tt.Assert.Equal("0.0000100", resource.Reserves[0].Amount)

	tt.Assert.Equal("USD:GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Reserves[1].Asset)
	tt.Assert.Equal("0.0000200", resource.Reserves[1].Amount)

	// try to fetch pool which does not exist
	_, err = handler.GetResource(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{},
		map[string]string{"id": "123816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		q,
	))
	tt.Assert.Error(err)
	tt.Assert.True(q.NoRows(errors.Cause(err)))

	// try to fetch a random invalid hex id
	_, err = handler.GetResource(httptest.NewRecorder(), makeRequest(
		t,
		map[string]string{},
		map[string]string{"id": "0000001112122"},
		q,
	))
	tt.Assert.Error(err)
	p := err.(*problem.P)
	tt.Assert.Equal("bad_request", p.Type)
	tt.Assert.Equal("id", p.Extras["invalid_field"])
	tt.Assert.Equal("0000001112122 does not validate as sha256", p.Extras["reason"])
}

func TestGetLiquidityPools(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}

	builder := q.NewLiquidityPoolsBatchInsertBuilder(2)
	err := builder.Add(tt.Ctx, history.LiquidityPool{
		PoolID:         "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            30,
		TrustlineCount: 100,
		ShareCount:     200,
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
	})
	tt.Assert.NoError(err)
	err = builder.Add(tt.Ctx, history.LiquidityPool{
		PoolID:         "d827bf10a721d217de3cd9ab3f10198a54de558c093a511ec426028618df2633",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            30,
		TrustlineCount: 300,
		ShareCount:     400,
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
	})
	tt.Assert.NoError(err)
	err = builder.Exec(tt.Ctx)
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
	tt.Assert.Equal(uint64(200), resource.TotalShares)

	tt.Assert.Equal("native", resource.Reserves[0].Asset)
	tt.Assert.Equal("0.0000100", resource.Reserves[0].Amount)

	tt.Assert.Equal("USD:GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", resource.Reserves[1].Asset)
	tt.Assert.Equal("0.0000200", resource.Reserves[1].Amount)

	resource = response[1].(protocol.LiquidityPool)
	tt.Assert.Equal("d827bf10a721d217de3cd9ab3f10198a54de558c093a511ec426028618df2633", resource.ID)
	tt.Assert.Equal("constant_product", resource.Type)
	tt.Assert.Equal(uint32(30), resource.FeeBP)
	tt.Assert.Equal(uint64(300), resource.TotalTrustlines)
	tt.Assert.Equal(uint64(400), resource.TotalShares)

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
