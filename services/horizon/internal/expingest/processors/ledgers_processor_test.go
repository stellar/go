package processors

import (
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type LedgersProcessorTestSuiteLedger struct {
	suite.Suite
	processor     *LedgersProcessor
	mockQ         *history.MockQLedgers
	header        xdr.LedgerHeaderHistoryEntry
	successCount  int
	failedCount   int
	opCount       int
	ingestVersion int
	txs           []io.LedgerTransaction
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
	op := xdr.BumpSequenceOp{BumpTo: 30000}
	for i := 0; i < numOps; i++ {
		operations = append(operations, xdr.Operation{
			Body: xdr.OperationBody{
				Type:           xdr.OperationTypeBumpSequence,
				BumpSequenceOp: &op,
			},
		})
	}
	sourceAID := xdr.MustAddress("GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY")
	return io.LedgerTransaction{
		Result: xdr.TransactionResultPair{
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					Code: code,
				},
			},
		},
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: sourceAID.ToMuxedAccount(),
					Operations:    operations,
				},
			},
		},
	}
}

func (s *LedgersProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQLedgers{}
	s.ingestVersion = 100
	s.header = xdr.LedgerHeaderHistoryEntry{
		Header: xdr.LedgerHeader{
			LedgerSeq: xdr.Uint32(20),
		},
	}
	s.processor = NewLedgerProcessor(
		s.mockQ,
		s.header,
		s.ingestVersion,
	)

	s.txs = []io.LedgerTransaction{
		createTransaction(true, 1),
		createTransaction(false, 3),
		createTransaction(true, 4),
	}

	s.successCount = 2
	s.failedCount = 1
	s.opCount = 5
}

func (s *LedgersProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertLedgerSucceeds() {
	s.mockQ.On(
		"InsertLedger",
		s.header,
		s.successCount,
		s.failedCount,
		s.opCount,
		s.ingestVersion,
	).Return(int64(1), nil)

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}

	err := s.processor.Commit()
	s.Assert().NoError(err)
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertLedgerReturnsError() {
	s.mockQ.On(
		"InsertLedger",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(int64(0), errors.New("transient error"))

	err := s.processor.Commit()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Could not insert ledger: transient error")
}

func (s *LedgersProcessorTestSuiteLedger) TestInsertLedgerNoRowsAffected() {
	s.mockQ.On(
		"InsertLedger",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(int64(0), nil)

	err := s.processor.Commit()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "0 rows affected when ingesting new ledger: 20")
}
