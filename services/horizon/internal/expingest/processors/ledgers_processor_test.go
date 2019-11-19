package processors

import (
	"context"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/verify"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

type LedgersProcessorTestSuiteLedger struct {
	suite.Suite
	processor        *DatabaseProcessor
	mockQ            *history.MockQLedgers
	mockLedgerReader *io.MockLedgerReader
	mockLedgerWriter *io.MockLedgerWriter
	header           xdr.LedgerHeaderHistoryEntry
	closeTime        int64
	successCount     int
	failedCount      int
	opCount          int
}

func TestLedgersProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(LedgersProcessorTestSuiteLedger))
}
func (s *LedgersProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQLedgers{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}

	s.processor = &DatabaseProcessor{
		Action:   Ledgers,
		LedgersQ: s.mockQ,
	}

	// Reader and Writer should be always closed and once
	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(io.Change{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()

	s.header = xdr.LedgerHeaderHistoryEntry{}
	s.closeTime = int64(1234)
	s.successCount = 2
	s.failedCount = 1
	s.opCount = 5
	s.mockLedgerReader.On("GetHeader").Return(s.header).Once()
	s.mockLedgerReader.On("CloseTime").Return(s.closeTime).Once()
	s.mockLedgerReader.On("SuccessfulTransactionCount").Return(s.successCount).Once()
	s.mockLedgerReader.On("FailedTransactionCount").Return(s.failedCount).Once()
	s.mockLedgerReader.On("SuccessfulLedgerOperationCount").Return(s.opCount).Once()
}

func (s *LedgersProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertLedgerSucceeds() {

	s.mockQ.On(
		"InsertLedger",
		s.header,
		s.closeTime,
		s.successCount,
		s.failedCount,
		s.opCount,
	).Return(int64(1), nil)

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertLedgerReturnsError() {
	s.mockQ.On(
		"InsertLedger",
		s.header,
		s.closeTime,
		s.successCount,
		s.failedCount,
		s.opCount,
	).Return(int64(0), errors.New("transient error"))

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Could not insert ledger: transient error")
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertLedgerNoRowsAffected() {
	s.mockLedgerReader.On("GetSequence").Return(uint32(1)).Once()

	s.mockQ.On(
		"InsertLedger",
		s.header,
		s.closeTime,
		s.successCount,
		s.failedCount,
		s.opCount,
	).Return(int64(0), nil)

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().IsType(verify.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "No rows affected when ingesting new ledger: 1")
}
