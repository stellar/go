package processors

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/suite"
)

type TransactionsProcessorTestSuiteLedger struct {
	suite.Suite
	processor              *TransactionProcessor
	mockQ                  *history.MockQTransactions
	mockBatchInsertBuilder *history.MockTransactionsBatchInsertBuilder
}

func TestTransactionsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(TransactionsProcessorTestSuiteLedger))
}

func (s *TransactionsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQTransactions{}
	s.mockBatchInsertBuilder = &history.MockTransactionsBatchInsertBuilder{}

	s.mockQ.
		On("NewTransactionBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.processor = NewTransactionProcessor(s.mockQ, 20)
}

func (s *TransactionsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *TransactionsProcessorTestSuiteLedger) TestAddTransactionsSucceeds() {
	sequence := uint32(20)

	firstTx := createTransaction(true, 1)
	secondTx := createTransaction(false, 3)
	thirdTx := createTransaction(true, 4)

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Add", secondTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Add", thirdTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.Assert().NoError(s.processor.Commit())

	err := s.processor.ProcessTransaction(firstTx)
	s.Assert().NoError(err)

	err = s.processor.ProcessTransaction(secondTx)
	s.Assert().NoError(err)

	err = s.processor.ProcessTransaction(thirdTx)
	s.Assert().NoError(err)
}

func (s *TransactionsProcessorTestSuiteLedger) TestAddTransactionsFails() {
	sequence := uint32(20)
	firstTx := createTransaction(true, 1)
	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).
		Return(errors.New("transient error")).Once()

	err := s.processor.ProcessTransaction(firstTx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error batch inserting transaction rows: transient error")
}

func (s *TransactionsProcessorTestSuiteLedger) TestExecFails() {
	sequence := uint32(20)
	firstTx := createTransaction(true, 1)

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()

	err := s.processor.ProcessTransaction(firstTx)
	s.Assert().NoError(err)

	err = s.processor.Commit()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error flushing transaction batch: transient error")
}
