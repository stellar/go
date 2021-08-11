//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package processors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func TestLiquidityPoolsChangeProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(LiquidityPoolsChangeProcessorTestSuiteState))
}

type LiquidityPoolsChangeProcessorTestSuiteState struct {
	suite.Suite
	ctx                    context.Context
	processor              *LiquidityPoolsChangeProcessor
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

	s.processor = NewLiquidityPoolsChangeProcessor(s.mockQ)
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
	lPool := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
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

	s.mockBatchInsertBuilder.On("Add", s.ctx, &xdr.LedgerEntry{
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPool,
		},
	}).Return(nil).Once()

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:          xdr.LedgerEntryTypeLiquidityPool,
				LiquidityPool: &lPool,
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
	processor              *LiquidityPoolsChangeProcessor
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

	s.processor = NewLiquidityPoolsChangeProcessor(s.mockQ)
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
	lPool := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
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
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPool,
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
		Post: &entry,
	})
	s.Assert().NoError(err)

	// add sponsor
	updated := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPool,
		},
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	entry.LastModifiedLedgerSeq = entry.LastModifiedLedgerSeq - 1
	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  &entry,
		Post: &updated,
	})
	s.Assert().NoError(err)

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockBatchInsertBuilder.On(
		"Add",
		s.ctx,
		&updated,
	).Return(nil).Once()
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) TestUpdateLiquidityPool() {
	lPool := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
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
			LiquidityPool: &lPool,
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
	updated := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type:          xdr.LedgerEntryTypeLiquidityPool,
			LiquidityPool: &lPool,
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
		Post: &updated,
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"UpdateLiquidityPool",
		s.ctx,
		updated,
	).Return(int64(1), nil).Once()
}

func (s *LiquidityPoolsChangeProcessorTestSuiteLedger) TestRemoveLiquidityPool() {
	lPool := xdr.LiquidityPoolEntry{
		LiquidityPoolId: xdr.PoolId{0xca, 0xfe, 0xba, 0xba, 0xbe, 0xde, 0xad, 0xbe, 0xef},
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
			LiquidityPool: &lPool,
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
		lPool,
	).Return(int64(1), nil).Once()
}
