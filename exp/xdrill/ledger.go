package xdrill

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

type Ledger struct {
	ledger *xdr.LedgerCloseMeta
}

func (l Ledger) Sequence() uint32 {
	return uint32(l.ledger.LedgerHeaderHistoryEntry().Header.LedgerSeq)
}

func (l Ledger) ID() int64 {
	return utils.NewID(int32(l.Sequence()), 0, 0).ToInt64()
}

func (l Ledger) Hash() string {
	return utils.HashToHexString(l.ledger.LedgerHeaderHistoryEntry().Hash)
}

func (l Ledger) PreviousHash() string {
	return utils.HashToHexString(l.ledger.PreviousLedgerHash())
}

func (l Ledger) CloseTime() int64 {
	return l.ledger.LedgerCloseTime()
}

func (l Ledger) ClosedAt() time.Time {
	return time.Unix(l.CloseTime(), 0).UTC()
}

func (l Ledger) TotalCoins() int64 {
	return int64(l.ledger.LedgerHeaderHistoryEntry().Header.TotalCoins)
}

func (l Ledger) FeePool() int64 {
	return int64(l.ledger.LedgerHeaderHistoryEntry().Header.FeePool)
}

func (l Ledger) BaseFee() uint32 {
	return uint32(l.ledger.LedgerHeaderHistoryEntry().Header.BaseFee)
}

func (l Ledger) BaseReserve() uint32 {
	return uint32(l.ledger.LedgerHeaderHistoryEntry().Header.BaseReserve)
}

func (l Ledger) MaxTxSetSize() uint32 {
	return uint32(l.ledger.LedgerHeaderHistoryEntry().Header.MaxTxSetSize)
}

func (l Ledger) LedgerVersion() uint32 {
	return uint32(l.ledger.LedgerHeaderHistoryEntry().Header.LedgerVersion)
}

func (l Ledger) SorobanFeeWrite1Kb() (int64, bool) {
	lcmV1, ok := l.ledger.GetV1()
	if !ok {
		return 0, false
	}

	extV1, ok := lcmV1.Ext.GetV1()
	if !ok {
		return 0, false
	}

	return int64(extV1.SorobanFeeWrite1Kb), true
}

func (l Ledger) TotalByteSizeOfBucketList() (uint64, bool) {
	lcmV1, ok := l.ledger.GetV1()
	if !ok {
		return 0, false
	}

	return uint64(lcmV1.TotalByteSizeOfBucketList), true
}

func (l Ledger) NodeID() (string, bool) {
	LedgerCloseValueSignature, ok := l.ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return "", false

	}
	nodeID, ok := utils.GetAddress(LedgerCloseValueSignature.NodeId)
	if !ok {
		return "", false
	}

	return nodeID, true
}

func (l Ledger) Signature() (string, bool) {
	LedgerCloseValueSignature, ok := l.ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return "", false
	}

	return base64.StdEncoding.EncodeToString(LedgerCloseValueSignature.Signature), true
}

// Add docstring to larger, more complicated functions
func (l Ledger) TransactionCounts() (successTxCount, failedTxCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := getTransactionSet(l)
	switch l.ledger.V {
	case 0:
		results = l.ledger.V0.TxProcessing
	case 1:
		results = l.ledger.V1.TxProcessing
	}
	txCount := len(transactions)
	if txCount != len(results) {
		return 0, 0, false
	}

	for i := 0; i < txCount; i++ {
		if results[i].Result.Successful() {
			successTxCount++
		} else {
			failedTxCount++
		}
	}

	return successTxCount, failedTxCount, true
}

// Add docstring to larger, more complicated functions
func (l Ledger) OperationCounts() (operationCount, txSetOperationCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := getTransactionSet(l)
	switch l.ledger.V {
	case 0:
		results = l.ledger.V0.TxProcessing
	case 1:
		results = l.ledger.V1.TxProcessing
	}

	txCount := len(transactions)
	if txCount != len(results) {
		return 0, 0, false
	}

	for i := 0; i < txCount; i++ {
		operations := transactions[i].Operations()
		numberOfOps := int32(len(operations))
		txSetOperationCount += numberOfOps

		// for successful transactions, the operation count is based on the operations results slice
		if results[i].Result.Successful() {
			operationResults, ok := results[i].Result.OperationResults()
			if !ok {
				return 0, 0, false
			}

			operationCount += int32(len(operationResults))
		}

	}

	return operationCount, txSetOperationCount, true
}

func getTransactionSet(l Ledger) (transactionProcessing []xdr.TransactionEnvelope) {
	switch l.ledger.V {
	case 0:
		return l.ledger.V0.TxSet.Txs
	case 1:
		switch l.ledger.V1.TxSet.V {
		case 0:
			return getTransactionPhase(l.ledger.V1.TxSet.V1TxSet.Phases)
		default:
			panic(fmt.Sprintf("unsupported LedgerCloseMeta.V1.TxSet.V: %d", l.ledger.V1.TxSet.V))
		}
	default:
		panic(fmt.Sprintf("unsupported LedgerCloseMeta.V: %d", l.ledger.V))
	}
}

func getTransactionPhase(transactionPhase []xdr.TransactionPhase) (transactionEnvelope []xdr.TransactionEnvelope) {
	transactionSlice := []xdr.TransactionEnvelope{}
	for _, phase := range transactionPhase {
		switch phase.V {
		case 0:
			components := phase.MustV0Components()
			for _, component := range components {
				switch component.Type {
				case 0:
					transactionSlice = append(transactionSlice, component.TxsMaybeDiscountedFee.Txs...)

				default:
					panic(fmt.Sprintf("unsupported TxSetComponentType: %d", component.Type))
				}

			}
		default:
			panic(fmt.Sprintf("unsupported TransactionPhase.V: %d", phase.V))
		}
	}

	return transactionSlice
}

func CreateSampleTx(sequence int64, operationCount int) xdr.TransactionEnvelope {
	kp, err := keypair.Random()
	PanicOnError(err)

	operations := []txnbuild.Operation{}
	operationType := &txnbuild.BumpSequence{
		BumpTo: 0,
	}
	for i := 0; i < operationCount; i++ {
		operations = append(operations, operationType)
	}

	sourceAccount := txnbuild.NewSimpleAccount(kp.Address(), int64(0))
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    operations,
			BaseFee:       txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		},
	)
	PanicOnError(err)

	env := tx.ToXDR()
	return env
}

// PanicOnError is a function that panics if the provided error is not nil
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
