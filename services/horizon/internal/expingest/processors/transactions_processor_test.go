package processors

import (
	"context"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/suite"
)

type TransactionsProcessorTestSuiteLedger struct {
	suite.Suite
	processor              *TransactionProcessor
	mockQ                  *history.MockQTransactions
	mockBatchInsertBuilder *history.MockTransactionsBatchInsertBuilder
	mockLedgerReader       *io.MockLedgerReader
	mockLedgerWriter       *io.MockLedgerWriter
	context                context.Context
}

func TestTransactionsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(TransactionsProcessorTestSuiteLedger))
}

func (s *TransactionsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQTransactions{}
	s.mockBatchInsertBuilder = &history.MockTransactionsBatchInsertBuilder{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}
	s.context = context.WithValue(context.Background(), IngestUpdateDatabase, true)

	s.processor = &TransactionProcessor{
		TransactionsQ: s.mockQ,
	}

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()
}

func (s *TransactionsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *TransactionsProcessorTestSuiteLedger) TestInsertExpLedgerIgnoredWhenNotDatabaseIngestion() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *TransactionsProcessorTestSuiteLedger) TestAddTransactionsSucceeds() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewTransactionBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(20)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)
	secondTx := createTransaction(false, 3)
	thirdTx := createTransaction(true, 4)

	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(secondTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(thirdTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Add", secondTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Add", thirdTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	s.mockQ.On("CheckExpTransactions", int32(sequence-10)).Return(true, nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *TransactionsProcessorTestSuiteLedger) TestAddTransactionsFails() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewTransactionBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(20)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)
	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).
		Return(errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error batch inserting transaction rows: transient error")
}

func (s *TransactionsProcessorTestSuiteLedger) TestExecFails() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewTransactionBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(20)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)
	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error flushing transaction batch: transient error")
}

func (s *TransactionsProcessorTestSuiteLedger) TestCheckExpTransactionsError() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewTransactionBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(20)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)

	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	s.mockQ.On("CheckExpTransactions", int32(sequence-10)).
		Return(false, errors.New("transient check exp ledger error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *TransactionsProcessorTestSuiteLedger) TestCheckExpTransactionsDoesNotMatch() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewTransactionBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(20)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)

	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(nil).Once()
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	s.mockQ.On("CheckExpTransactions", int32(sequence-10)).
		Return(false, nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}
