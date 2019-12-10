package processors

import (
	"context"
	"database/sql"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
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
	context          context.Context
	header           xdr.LedgerHeaderHistoryEntry
	successCount     int
	failedCount      int
	opCount          int
	ingestVersion    int
}

func TestLedgersProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(LedgersProcessorTestSuiteLedger))
}

func createTransaction(successful bool, numOps int) io.LedgerTransaction {
	code := xdr.TransactionResultCodeTxSuccess
	if !successful {
		code = xdr.TransactionResultCodeTxFailed
	}

	operations := []xdr.Operation{}
	for i := 0; i < numOps; i++ {
		operations = append(operations, xdr.Operation{})
	}

	return io.LedgerTransaction{
		Result: xdr.TransactionResultPair{
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					Code: code,
				},
			},
		},
		Envelope: xdr.TransactionEnvelope{
			Tx: xdr.Transaction{
				Operations: operations,
			},
		},
	}
}

func (s *LedgersProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQLedgers{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}
	s.ingestVersion = 100
	s.context = context.WithValue(context.Background(), IngestUpdateDatabase, true)

	s.processor = &DatabaseProcessor{
		Action:        Ledgers,
		LedgersQ:      s.mockQ,
		IngestVersion: s.ingestVersion,
	}

	s.mockLedgerReader.On("GetSequence").Return(uint32(20)).Maybe()

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(io.Change{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Read").
		Return(createTransaction(true, 1), nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(createTransaction(false, 3), nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(createTransaction(true, 4), nil).Once()
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
	s.successCount = 2
	s.failedCount = 1
	s.opCount = 5
	s.mockLedgerReader.On("GetHeader").Return(s.header).Once()
}

func (s *LedgersProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertExpLedgerIgnoredWhenNotDatabaseIngestion() {
	// Clear mockLedgerReader expectations
	s.mockLedgerReader = &io.MockLedgerReader{}

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(io.Change{}, stdio.EOF).Once()

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

func (s *LedgersProcessorTestSuiteLedger) TestInsertExpLedgerSucceeds() {
	s.mockQ.On(
		"InsertExpLedger",
		s.header,
		s.successCount,
		s.failedCount,
		s.opCount,
		s.ingestVersion,
	).Return(int64(1), nil)
	s.mockQ.On("CheckExpLedger", int32(10)).Return(true, nil)

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *LedgersProcessorTestSuiteLedger) TestCheckExpLedgerNotFound() {
	s.mockQ.On(
		"InsertExpLedger",
		s.header,
		s.successCount,
		s.failedCount,
		s.opCount,
		s.ingestVersion,
	).Return(int64(1), nil)
	s.mockQ.On("CheckExpLedger", int32(10)).Return(false, sql.ErrNoRows)

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *LedgersProcessorTestSuiteLedger) TestCheckExpLedgerError() {
	s.mockQ.On(
		"InsertExpLedger",
		s.header,
		s.successCount,
		s.failedCount,
		s.opCount,
		s.ingestVersion,
	).Return(int64(1), nil)
	s.mockQ.On("CheckExpLedger", int32(10)).
		Return(false, errors.New("transient check exp ledger error"))

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Could not compare ledger 10: transient check exp ledger error")
}

func (s *LedgersProcessorTestSuiteLedger) TestCheckExpLedgerDoesNotMatch() {
	s.mockQ.On(
		"InsertExpLedger",
		s.header,
		s.successCount,
		s.failedCount,
		s.opCount,
		s.ingestVersion,
	).Return(int64(1), nil)
	s.mockQ.On("CheckExpLedger", int32(10)).
		Return(false, nil)

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(
		err,
		"ledger 10 in exp_history_ledgers does not match ledger in history_ledgers",
	)
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertExpLedgerReturnsError() {
	s.mockQ.On(
		"InsertExpLedger",
		s.header,
		s.successCount,
		s.failedCount,
		s.opCount,
		s.ingestVersion,
	).Return(int64(0), errors.New("transient error"))

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Could not insert ledger: transient error")
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertExpLedgerNoRowsAffected() {
	s.mockQ.On(
		"InsertExpLedger",
		s.header,
		s.successCount,
		s.failedCount,
		s.opCount,
		s.ingestVersion,
	).Return(int64(0), nil)

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "No rows affected when ingesting new ledger: 20")
}
