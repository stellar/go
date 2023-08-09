//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package processors

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/guregu/null"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type OperationsProcessorTestSuiteLedger struct {
	suite.Suite
	ctx                    context.Context
	processor              *OperationProcessor
	mockSession            *db.MockSession
	mockBatchInsertBuilder *history.MockOperationsBatchInsertBuilder
}

func TestOperationProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(OperationsProcessorTestSuiteLedger))
}

func (s *OperationsProcessorTestSuiteLedger) SetupTest() {
	s.ctx = context.Background()
	s.mockBatchInsertBuilder = &history.MockOperationsBatchInsertBuilder{}

	s.processor = NewOperationProcessor(
		s.mockBatchInsertBuilder,
	)
}

func (s *OperationsProcessorTestSuiteLedger) TearDownTest() {
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *OperationsProcessorTestSuiteLedger) mockBatchInsertAdds(txs []ingest.LedgerTransaction, sequence uint32) error {
	for _, t := range txs {
		for i, op := range t.Envelope.Operations() {
			expected := transactionOperationWrapper{
				index:          uint32(i),
				transaction:    t,
				operation:      op,
				ledgerSequence: sequence,
			}
			details, err := expected.Details()
			if err != nil {
				return err
			}
			detailsJSON, err := json.Marshal(details)
			if err != nil {
				return err
			}

			source := expected.SourceAccount()
			acID := source.ToAccountId()
			var muxedAccount null.String
			if source.Type == xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
				muxedAccount = null.StringFrom(source.Address())
			}
			s.mockBatchInsertBuilder.On(
				"Add",
				expected.ID(),
				expected.TransactionID(),
				expected.Order(),
				expected.OperationType(),
				detailsJSON,
				acID.Address(),
				muxedAccount,
			).Return(nil).Once()
		}
	}

	return nil
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationSucceeds() {
	sequence := uint32(56)
	lcm := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}

	unmuxed := xdr.MustAddress("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")
	muxed := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xdeadbeefdeadbeef,
			Ed25519: *unmuxed.Ed25519,
		},
	}
	firstTx := createTransaction(true, 1)
	firstTx.Index = 1
	firstTx.Envelope.Operations()[0].Body = xdr.OperationBody{
		Type: xdr.OperationTypePayment,
		PaymentOp: &xdr.PaymentOp{
			Destination: muxed,
			Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
			Amount:      100,
		},
	}
	firstTx.Envelope.V1.Tx.SourceAccount = muxed
	secondTx := createTransaction(false, 3)
	thirdTx := createTransaction(true, 4)

	txs := []ingest.LedgerTransaction{
		firstTx,
		secondTx,
		thirdTx,
	}

	var err error

	err = s.mockBatchInsertAdds(txs, sequence)
	s.Assert().NoError(err)
	s.mockBatchInsertBuilder.On("Exec", s.ctx, s.mockSession).Return(nil).Once()

	for _, tx := range txs {
		err = s.processor.ProcessTransaction(lcm, tx)
		s.Assert().NoError(err)
	}
	s.Assert().NoError(s.processor.Flush(s.ctx, s.mockSession))
}

func (s *OperationsProcessorTestSuiteLedger) TestAddOperationFails() {
	sequence := uint32(56)
	lcm := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}
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
			mock.Anything,
		).Return(errors.New("transient error")).Once()

	err := s.processor.ProcessTransaction(lcm, tx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error batch inserting operation rows: transient error")
}

func (s *OperationsProcessorTestSuiteLedger) TestExecFails() {
	sequence := uint32(56)
	lcm := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}
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
			mock.Anything,
		).Return(nil).Once()
	s.Assert().NoError(s.processor.ProcessTransaction(lcm, tx))

	s.mockBatchInsertBuilder.On("Exec", s.ctx, s.mockSession).Return(errors.New("transient error")).Once()
	err := s.processor.Flush(s.ctx, s.mockSession)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "transient error")
}
