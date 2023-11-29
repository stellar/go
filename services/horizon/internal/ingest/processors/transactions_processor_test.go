//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package processors

import (
	"context"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/suite"
)

type TransactionsProcessorTestSuiteLedger struct {
	suite.Suite
	ctx                    context.Context
	processor              *TransactionProcessor
	mockSession            *db.MockSession
	mockBatchInsertBuilder *history.MockTransactionsBatchInsertBuilder
}

func TestTransactionsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(TransactionsProcessorTestSuiteLedger))
}

func (s *TransactionsProcessorTestSuiteLedger) SetupTest() {
	s.ctx = context.Background()
	s.mockBatchInsertBuilder = &history.MockTransactionsBatchInsertBuilder{}
	s.processor = NewTransactionProcessor(s.mockBatchInsertBuilder)
}

func (s *TransactionsProcessorTestSuiteLedger) TearDownTest() {
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *TransactionsProcessorTestSuiteLedger) TestAddTransactionsSucceeds() {
	sequence := uint32(20)
	lcm := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}
	firstTx := createTransaction(true, 1)
	secondTx := createTransaction(false, 3)
	thirdTx := createTransaction(true, 4)

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Add", secondTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Add", thirdTx, sequence+1).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec", s.ctx, s.mockSession).Return(nil).Once()

	s.Assert().NoError(s.processor.ProcessTransaction(lcm, firstTx))
	s.Assert().NoError(s.processor.ProcessTransaction(lcm, secondTx))
	lcm.V0.LedgerHeader.Header.LedgerSeq++
	s.Assert().NoError(s.processor.ProcessTransaction(lcm, thirdTx))

	s.Assert().NoError(s.processor.Flush(s.ctx, s.mockSession))
}

func (s *TransactionsProcessorTestSuiteLedger) TestAddTransactionsFails() {
	sequence := uint32(20)
	lcm := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}
	firstTx := createTransaction(true, 1)
	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).
		Return(errors.New("transient error")).Once()

	err := s.processor.ProcessTransaction(lcm, firstTx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error batch inserting transaction rows: transient error")
}

func (s *TransactionsProcessorTestSuiteLedger) TestExecFails() {
	sequence := uint32(20)
	lcm := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}
	firstTx := createTransaction(true, 1)

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec", s.ctx, s.mockSession).Return(errors.New("transient error")).Once()

	s.Assert().NoError(s.processor.ProcessTransaction(lcm, firstTx))

	err := s.processor.Flush(s.ctx, s.mockSession)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "transient error")
}
