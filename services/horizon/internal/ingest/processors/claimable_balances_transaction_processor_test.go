//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package processors

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/xdr"
)

type ClaimableBalancesTransactionProcessorTestSuiteLedger struct {
	suite.Suite
	processor                         *ClaimableBalancesTransactionProcessor
	mockQ                             *history.MockQHistoryClaimableBalances
	mockTransactionBatchInsertBuilder *history.MockTransactionClaimableBalanceBatchInsertBuilder
	mockOperationBatchInsertBuilder   *history.MockOperationClaimableBalanceBatchInsertBuilder

	sequence uint32
}

func TestClaimableBalancesTransactionProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(ClaimableBalancesTransactionProcessorTestSuiteLedger))
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQHistoryClaimableBalances{}
	s.mockTransactionBatchInsertBuilder = &history.MockTransactionClaimableBalanceBatchInsertBuilder{}
	s.mockOperationBatchInsertBuilder = &history.MockOperationClaimableBalanceBatchInsertBuilder{}
	s.sequence = 20

	s.processor = NewClaimableBalancesTransactionProcessor(
		s.mockQ,
		s.sequence,
	)
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockTransactionBatchInsertBuilder.AssertExpectations(s.T())
	s.mockOperationBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) mockTransactionBatchAdd(transactionID, internalID int64, err error) {
	s.mockTransactionBatchInsertBuilder.On("Add", transactionID, internalID).Return(err).Once()
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) mockOperationBatchAdd(operationID, internalID int64, err error) {
	s.mockOperationBatchInsertBuilder.On("Add", operationID, internalID).Return(err).Once()
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) TestEmptyClaimableBalances() {
	// What is this expecting? Doesn't seem to assert anything meaningful...
	err := s.processor.Commit()
	s.Assert().NoError(err)
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) testOperationInserts(balanceID xdr.ClaimableBalanceId, body xdr.OperationBody) {
	// Setup the transaction
	internalID := int64(1234)
	txn := createTransaction(true, 1)
	txn.Envelope.Operations()[0].Body = body

	if body.Type == xdr.OperationTypeCreateClaimableBalance {
		// For insert test
		txn.Result.Result.Result.Results =
			&[]xdr.OperationResult{
				{
					Code: xdr.OperationResultCodeOpInner,
					Tr: &xdr.OperationResultTr{
						Type: xdr.OperationTypeCreateClaimableBalance,
						CreateClaimableBalanceResult: &xdr.CreateClaimableBalanceResult{
							Code:      xdr.CreateClaimableBalanceResultCodeCreateClaimableBalanceSuccess,
							BalanceId: &balanceID,
						},
					},
				},
			}
	}
	txnID := toid.New(int32(s.sequence), int32(txn.Index), 0).ToInt64()
	opID := (&transactionOperationWrapper{
		index:          uint32(0),
		transaction:    txn,
		operation:      txn.Envelope.Operations()[0],
		ledgerSequence: s.sequence,
	}).ID()

	hexID, _ := xdr.MarshalHex(balanceID)

	// Setup a q
	s.mockQ.On("CreateHistoryClaimableBalances", mock.AnythingOfType("[]xdr.ClaimableBalanceId"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]xdr.ClaimableBalanceId)
			s.Assert().ElementsMatch(
				[]xdr.ClaimableBalanceId{
					balanceID,
				},
				arg,
			)
		}).Return(map[string]int64{
		hexID: internalID,
	}, nil).Once()

	// Prepare to process transactions successfully
	s.mockQ.On("NewTransactionClaimableBalanceBatchInsertBuilder", maxBatchSize).
		Return(s.mockTransactionBatchInsertBuilder).Once()
	s.mockTransactionBatchAdd(txnID, internalID, nil)
	s.mockTransactionBatchInsertBuilder.On("Exec").Return(nil).Once()

	// Prepare to process operations successfully
	s.mockQ.On("NewOperationClaimableBalanceBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationBatchInsertBuilder).Once()
	s.mockOperationBatchAdd(opID, internalID, nil)
	s.mockOperationBatchInsertBuilder.On("Exec").Return(nil).Once()

	// Process the transaction
	err := s.processor.ProcessTransaction(txn)
	s.Assert().NoError(err)
	err = s.processor.Commit()
	s.Assert().NoError(err)
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) TestIngestClaimableBalancesInsertsClaimClaimableBalance() {
	balanceID := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3},
	}
	s.testOperationInserts(balanceID, xdr.OperationBody{
		Type: xdr.OperationTypeClaimClaimableBalance,
		ClaimClaimableBalanceOp: &xdr.ClaimClaimableBalanceOp{
			BalanceId: balanceID,
		},
	})
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) TestIngestClaimableBalancesInsertsClawbackClaimableBalance() {
	balanceID := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3},
	}
	s.testOperationInserts(balanceID, xdr.OperationBody{
		Type: xdr.OperationTypeClawbackClaimableBalance,
		ClawbackClaimableBalanceOp: &xdr.ClawbackClaimableBalanceOp{
			BalanceId: balanceID,
		},
	})
}

func (s *ClaimableBalancesTransactionProcessorTestSuiteLedger) TestIngestClaimableBalancesInsertsCreateClaimableBalance() {
	balanceID := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &xdr.Hash{1, 2, 3},
	}
	s.testOperationInserts(balanceID, xdr.OperationBody{
		Type:                     xdr.OperationTypeCreateClaimableBalance,
		CreateClaimableBalanceOp: &xdr.CreateClaimableBalanceOp{},
	})
}
