//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package processors

import (
	"context"
	"testing"

	"github.com/stellar/go/gxdr"
	"github.com/stellar/go/randxdr"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/db"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func TestFuzzLiquidityPools(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{&db.Session{DB: tt.HorizonDB}}
	pp := NewLiquidityPoolsProcessor(q)
	gen := randxdr.NewGenerator()

	var changes []xdr.LedgerEntryChange
	for i := 0; i < 1000; i++ {
		change := xdr.LedgerEntryChange{}
		shape := &gxdr.LedgerEntryChange{}
		gen.Next(
			shape,
			[]randxdr.Preset{
				{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
				{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32()},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.params.fee"), randxdr.SetPositiveNum32()},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.reserveA"), randxdr.SetPositiveNum64()},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.reserveB"), randxdr.SetPositiveNum64()},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.totalPoolShares"), randxdr.SetPositiveNum64()},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.poolSharesTrustLineCount"), randxdr.SetPositiveNum64()},
			},
		)
		tt.Assert.NoError(gxdr.Convert(shape, &change))
		changes = append(changes, change)
	}

	for _, change := range ingest.GetChangesFromLedgerEntryChanges(changes) {
		tt.Assert.NoError(pp.ProcessChange(tt.Ctx, change))
	}

	tt.Assert.NoError(pp.Commit(tt.Ctx))
}
func TestLiquidityPoolsChangeProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(LiquidityPoolsChangeProcessorTestSuiteState))
}

type LiquidityPoolsChangeProcessorTestSuiteState struct {
	suite.Suite
	ctx                    context.Context
	processor              *LiquidityPoolsProcessor
	mockQ                  *history.MockQLiquidityPools
	mockBatchInsertBuilder *history.MockLiquidityPoolsBatchInsertBuilder
}

func (s *LiquidityPoolsChangeProcessorTestSuiteState) SetupTest() {
	s.ctx = context.Background()
	s.mockQ = &history.MockQLiquidityPools{}
	s.mockBatchInsertBuilder = &history.MockLiquidityPoolsBatchInsertBuilder{}

	s.mockQ.
		On("NewLiquidityPoolsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder)

	s.processor = NewLiquidityPoolsProcessor(s.mockQ)
}

func (s *LiquidityPoolsChangeProcessorTestSuiteState) TearDownTest() {
	s.mockBatchInsertBuilder.On("Exec", s.ctx).Return(nil).Once()
	s.Assert().NoError(s.processor.Commit(s.ctx))
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *LiquidityPoolsChangeProcessorTestSuiteState) TestNoEntries() {
	// Nothing processed, assertions in TearDownTest.
}

func (s *LiquidityPoolsChangeProcessorTestSuiteState) TestCreatesLiquidityPools() {
	lastModifiedLedgerSeq := xdr.Uint32(123)
	lpoolEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 500,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	lp := history.LiquidityPool{
		PoolID:         "cafebabedeadbeef000000000000000000000000000000000000000000000000",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            34,
		TrustlineCount: 52115,
		ShareCount:     412241,
		AssetReserves: []history.LiquidityPoolAssetReserve{
			{
				xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				450,
			},
			{
				xdr.MustNewNativeAsset(),
				500,
			},
		},
		LastModifiedLedger: 123,
	}

	s.mockBatchInsertBuilder.On("Add", s.ctx, lp).Return(nil).Once()

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:          xdr.LedgerEntryTypeLiquidityPool,
				LiquidityPool: &lpoolEntry,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)
}

func TestLiquidityPoolsChangeProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(LiquidityPoolsChangeProcessorTestSuiteLedger))
}

type LiquidityPoolsChangeProcessorTestSuiteLedger struct {
	suite.Suite
	ctx                    context.Context
	processor              *LiquidityPoolsProcessor
	mockQ                  *history.MockQLiquidityPools
	mockBatchInsertBuilder *history.MockLiquidityPoolsBatchInsertBuilder
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) SetupTest() {
	s.ctx = context.Background()
	s.mockQ = &history.MockQLiquidityPools{}
	s.mockBatchInsertBuilder = &history.MockLiquidityPoolsBatchInsertBuilder{}

	s.mockQ.
		On("NewLiquidityPoolsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder)

	s.processor = NewLiquidityPoolsProcessor(s.mockQ)
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) TearDownTest() {
	s.mockBatchInsertBuilder.On("Exec", s.ctx).Return(nil).Once()
	s.Assert().NoError(s.processor.Commit(s.ctx))
	s.mockQ.AssertExpectations(s.T())
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) TestNoTransactions() {
	// Nothing processed, assertions in TearDownTest.
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) TestNewLiquidityPool() {
	lastModifiedLedgerSeq := xdr.Uint32(123)
	lpEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 500,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	pre := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lpEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: nil,
			},
		},
	}
	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  nil,
		Post: &pre,
	})
	s.Assert().NoError(err)

	// add sponsor
	post := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lpEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	pre.LastModifiedLedgerSeq = pre.LastModifiedLedgerSeq - 1
	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  &pre,
		Post: &post,
	})
	s.Assert().NoError(err)

	postLP := history.LiquidityPool{
		PoolID:         "cafebabedeadbeef000000000000000000000000000000000000000000000000",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            34,
		TrustlineCount: 52115,
		ShareCount:     412241,
		AssetReserves: []history.LiquidityPoolAssetReserve{
			{
				xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				450,
			},
			{
				xdr.MustNewNativeAsset(),
				500,
			},
		},
		LastModifiedLedger: 123,
	}
	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockBatchInsertBuilder.On(
		"Add",
		s.ctx,
		postLP,
	).Return(nil).Once()
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) TestUpdateLiquidityPool() {
	lpEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    34,
				},
				ReserveA:                 450,
				ReserveB:                 500,
				TotalPoolShares:          412241,
				PoolSharesTrustLineCount: 52115,
			},
		},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	pre := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lpEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: nil,
			},
		},
	}

	// add sponsor
	post := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lpEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  &pre,
		Post: &post,
	})
	s.Assert().NoError(err)

	postLP := history.LiquidityPool{
		PoolID:         "cafebabedeadbeef000000000000000000000000000000000000000000000000",
		Type:           xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
		Fee:            34,
		TrustlineCount: 52115,
		ShareCount:     412241,
		AssetReserves: []history.LiquidityPoolAssetReserve{
			{
				xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				450,
			},
			{
				xdr.MustNewNativeAsset(),
				500,
			},
		},
		LastModifiedLedger: 123,
	}

	s.mockQ.On(
		"UpdateLiquidityPool",
		s.ctx,
		postLP,
	).Return(int64(1), nil).Once()
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) TestRemoveLiquidityPool() {
	lpEntry := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
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
	lastModifiedLedgerSeq := xdr.Uint32(123)
	pre := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lpEntry,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: nil,
			},
		},
	}
	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  &pre,
		Post: nil,
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"RemoveLiquidityPool",
		s.ctx,
		"cafebabedeadbeef000000000000000000000000000000000000000000000000",
	).Return(int64(1), nil).Once()
}
