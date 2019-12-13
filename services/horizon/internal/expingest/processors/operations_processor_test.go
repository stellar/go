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

type OperationsProcessorTestSuiteLedger struct {
	suite.Suite
	processor              *OperationProcessor
	mockQ                  *history.MockQOperations
	mockBatchInsertBuilder *history.MockOperationsBatchInsertBuilder
	mockLedgerReader       *io.MockLedgerReader
	mockLedgerWriter       *io.MockLedgerWriter
	context                context.Context
}

func TestOperationProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(OperationsProcessorTestSuiteLedger))
}

func (s *OperationsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQOperations{}
	s.mockBatchInsertBuilder = &history.MockOperationsBatchInsertBuilder{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}
	s.context = context.WithValue(context.Background(), IngestUpdateDatabase, true)

	s.processor = &OperationProcessor{
		OperationsQ: s.mockQ,
	}

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()
}

func (s *OperationsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *OperationsProcessorTestSuiteLedger) TestInsertExpLedgerIgnoredWhenNotDatabaseIngestion() {
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

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationSucceeds() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewOperationBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(56)
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

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationFails() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewOperationBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(56)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)
	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockBatchInsertBuilder.On("Add", firstTx, sequence).Return(errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error batch inserting operation rows: transient error")
}

func (s *OperationsProcessorTestSuiteLedger) TestExecFails() {
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockQ.
		On("NewOperationBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	sequence := uint32(20)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error flushing operation batch: transient error")
}
