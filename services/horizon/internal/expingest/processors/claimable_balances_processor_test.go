package processors

import (
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestClaimableBalancesProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(ClaimableBalancesProcessorTestSuiteState))
}

type ClaimableBalancesProcessorTestSuiteState struct {
	suite.Suite
	processor        *ClaimableBalancesProcessor
	mockQ            *history.MockQClaimableBalances
	mockBatchBuilder *history.MockClaimableBalancesBatchInsertBuilder
}

func (s *ClaimableBalancesProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQClaimableBalances{}
	s.mockBatchBuilder = &history.MockClaimableBalancesBatchInsertBuilder{}

	s.mockQ.
		On("NewClaimableBalancesBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchBuilder)

	s.processor = NewClaimableBalancesProcessor(s.mockQ)
}

func (s *ClaimableBalancesProcessorTestSuiteState) TearDownTest() {
	s.mockBatchBuilder.On("Exec").Return(nil).Once()
	s.Assert().NoError(s.processor.Commit())
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchBuilder.AssertExpectations(s.T())
}

func (s *ClaimableBalancesProcessorTestSuiteState) TestNoEntries() {
	// Nothing processed, assertions in TearDownTest.
}

func (s *ClaimableBalancesProcessorTestSuiteState) TestCreatesClaimableBalances() {
	lastModifiedLedgerSeq := xdr.Uint32(123)
	cBalance := xdr.ClaimableBalanceEntry{
		BalanceId: xdr.ClaimableBalanceId{
			Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
			V0:   &xdr.Hash{1, 2, 3},
		},
		Claimants: []xdr.Claimant{},
		Asset:     xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Amount:    10,
	}

	s.mockBatchBuilder.On("Add", &xdr.LedgerEntry{
		LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		Data: xdr.LedgerEntryData{
			Type:             xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &cBalance,
		},
	}).Return(nil).Once()

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &cBalance,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)
}
