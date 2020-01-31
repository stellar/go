package processors

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
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
}

func TestOperationProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(OperationsProcessorTestSuiteLedger))
}

func (s *OperationsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQOperations{}
	s.mockBatchInsertBuilder = &history.MockOperationsBatchInsertBuilder{}
	s.mockQ.
		On("NewOperationBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.processor = NewOperationProcessor(
		s.mockQ,
		56,
	)
}

func (s *OperationsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
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

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationSucceeds() {
	firstTx := createTransaction(true, 1)
	secondTx := createTransaction(false, 3)
	thirdTx := createTransaction(true, 4)

	txs := []io.LedgerTransaction{
		firstTx,
		secondTx,
		thirdTx,
	}

	var err error

	err = s.mockBatchInsertAdds(txs, uint32(56))
	s.Assert().NoError(err)
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.Assert().NoError(s.processor.Commit())

	for _, tx := range txs {
		err = s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationFails() {
	tx := createTransaction(true, 1)

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

	err := s.processor.ProcessTransaction(tx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error batch inserting operation rows: transient error")
}

func (s *OperationsProcessorTestSuiteLedger) TestExecFails() {
	s.mockBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()
	err := s.processor.Commit()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "transient error")
}
