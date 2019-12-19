package processors

import (
	"context"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/suite"
)

type TransactionsFilterProcessorTestSuiteLedger struct {
	suite.Suite
	processor              *TransactionFilterProcessor
	mockLedgerReader       *io.MockLedgerReader
	mockLedgerWriter       *io.MockLedgerWriter
	successfulTransactions []io.LedgerTransaction
	failedTransactions     []io.LedgerTransaction
}

func TestTransactionsFilterProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(TransactionsFilterProcessorTestSuiteLedger))
}

func (s *TransactionsFilterProcessorTestSuiteLedger) SetupTest() {
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}

	s.processor = &TransactionFilterProcessor{}

	s.successfulTransactions = []io.LedgerTransaction{
		createTransaction(true, 1),
		createTransaction(true, 4),
	}
	s.failedTransactions = []io.LedgerTransaction{
		createTransaction(false, 3),
	}

	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.successfulTransactions[0], nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.failedTransactions[0], nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.successfulTransactions[1], nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()
}

func (s *TransactionsFilterProcessorTestSuiteLedger) TearDownTest() {
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *TransactionsFilterProcessorTestSuiteLedger) TestIngestAllTransactions() {
	s.mockLedgerWriter.On("Write", s.successfulTransactions[0]).Return(nil).Once()
	s.mockLedgerWriter.On("Write", s.failedTransactions[0]).Return(nil).Once()
	s.mockLedgerWriter.On("Write", s.successfulTransactions[1]).Return(nil).Once()

	s.processor.IngestFailedTransactions = true
	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *TransactionsFilterProcessorTestSuiteLedger) TestIngestOnlySuccessfulTransactions() {
	s.mockLedgerWriter.On("Write", s.successfulTransactions[0]).Return(nil).Once()
	s.mockLedgerWriter.On("Write", s.successfulTransactions[1]).Return(nil).Once()

	s.processor.IngestFailedTransactions = false
	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *TransactionsFilterProcessorTestSuiteLedger) TestWritePipeError() {
	// Clear mockLedgerReader expectations
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.successfulTransactions[0], nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.failedTransactions[0], nil).Once()
	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.On("Write", s.successfulTransactions[0]).Return(nil).Once()
	s.mockLedgerWriter.On("Write", s.failedTransactions[0]).Return(stdio.ErrClosedPipe).Once()

	s.processor.IngestFailedTransactions = true
	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *TransactionsFilterProcessorTestSuiteLedger) TestWriteError() {
	// Clear mockLedgerReader expectations
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.successfulTransactions[0], nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.failedTransactions[0], nil).Once()
	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.On("Write", s.successfulTransactions[0]).Return(nil).Once()
	s.mockLedgerWriter.On("Write", s.failedTransactions[0]).
		Return(errors.New("transient write error")).Once()

	s.processor.IngestFailedTransactions = true
	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().EqualError(err, "transient write error")
}

func (s *TransactionsFilterProcessorTestSuiteLedger) TestReadError() {
	// Clear mockLedgerReader expectations
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.successfulTransactions[0], nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.failedTransactions[0], errors.New("transient read error")).Once()
	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.On("Write", s.successfulTransactions[0]).Return(nil).Once()

	s.processor.IngestFailedTransactions = true
	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().EqualError(err, "transient read error")
}
