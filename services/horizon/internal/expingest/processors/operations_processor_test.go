package processors

import (
	"context"
	"encoding/json"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/mock"
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

	s.mockLedgerWriter.On("Close").Return(nil).Once()
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	s.mockLedgerReader.On("Close").Return(nil).Once()

	s.mockQ.
		On("NewOperationBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
}

func (s *OperationsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *OperationsProcessorTestSuiteLedger) mockBatchInsertAdds(txs []io.LedgerTransaction, sequence uint32) error {
	for _, t := range txs {
		for i, op := range t.Envelope.Tx.Operations {
			expected := transactionOperationWrapper{
				index:          uint32(i),
				transaction:    t,
				operation:      op,
				ledgerSequence: sequence,
			}

			detailsJSON, err := json.Marshal(expected.Details())
			if err != nil {
				return err
			}

			s.mockBatchInsertBuilder.On(
				"Add",
				expected.ID(),
				expected.TransactionID(),
				expected.Order(),
				expected.OperationType(),
				detailsJSON,
				expected.SourceAccount().Address(),
			).Return(nil).Once()
		}
	}

	return nil
}

func (s *OperationsProcessorTestSuiteLedger) TestInsertExpLedgerIgnoredWhenNotDatabaseIngestion() {
	s.mockQ = &history.MockQOperations{}
	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationSucceeds() {
	sequence := uint32(56)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)
	secondTx := createTransaction(false, 3)
	thirdTx := createTransaction(true, 4)

	txs := []io.LedgerTransaction{
		firstTx,
		secondTx,
		thirdTx,
	}

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

	s.mockQ.
		On("CheckExpOperations", int32(sequence-10)).
		Return(true, nil).Once()

	var err error

	err = s.mockBatchInsertAdds(txs, sequence)
	s.Assert().NoError(err)

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	err = s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationFails() {
	sequence := uint32(56)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)
	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()

	s.mockBatchInsertBuilder.
		On(
			"Add",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(errors.New("transient error")).Once()

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
	sequence := uint32(56)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockBatchInsertBuilder.On("Exec").
		Return(errors.New("transient error")).
		Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error flushing operation batch: transient error")
}

func (s *OperationsProcessorTestSuiteLedger) TestCheckExpOperationsError() {
	sequence := uint32(56)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)

	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockBatchInsertAdds([]io.LedgerTransaction{firstTx}, sequence)
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	s.mockQ.
		On("CheckExpOperations", int32(sequence-10)).
		Return(false, errors.New("transient check exp ledger error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *OperationsProcessorTestSuiteLedger) TestCheckExpOperationsDoesNotMatch() {
	sequence := uint32(56)
	s.mockLedgerReader.On("GetSequence").Return(sequence).Once()

	firstTx := createTransaction(true, 1)

	s.mockLedgerReader.
		On("Read").
		Return(firstTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockBatchInsertAdds([]io.LedgerTransaction{firstTx}, sequence)
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	s.mockQ.On("CheckExpOperations", int32(sequence-10)).
		Return(false, nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}
