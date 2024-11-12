package ledger

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/xdr"
)

type Ledger struct {
	xdr.LedgerCloseMeta
}

func (l Ledger) Sequence() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.LedgerSeq)
}

func (l Ledger) ID() int64 {
	return utils.NewID(int32(l.LedgerSequence()), 0, 0).ToInt64()
}

func (l Ledger) Hash() string {
	return utils.HashToHexString(l.LedgerHeaderHistoryEntry().Hash)
}

func (l Ledger) PreviousHash() string {
	return utils.HashToHexString(l.PreviousLedgerHash())
}

func (l Ledger) CloseTime() int64 {
	return l.LedgerCloseTime()
}

func (l Ledger) ClosedAt() time.Time {
	return time.Unix(l.CloseTime(), 0).UTC()
}

func (l Ledger) TotalCoins() int64 {
	return int64(l.LedgerHeaderHistoryEntry().Header.TotalCoins)
}

func (l Ledger) FeePool() int64 {
	return int64(l.LedgerHeaderHistoryEntry().Header.FeePool)
}

func (l Ledger) BaseFee() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.BaseFee)
}

func (l Ledger) BaseReserve() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.BaseReserve)
}

func (l Ledger) MaxTxSetSize() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.MaxTxSetSize)
}

func (l Ledger) LedgerVersion() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.LedgerVersion)
}

func (l Ledger) SorobanFeeWrite1Kb() (int64, bool) {
	lcmV1, ok := l.GetV1()
	if ok {
		extV1, ok := lcmV1.Ext.GetV1()
		if ok {
			return int64(extV1.SorobanFeeWrite1Kb), true
		}
	}

	return 0, false
}

func (l Ledger) TotalByteSizeOfBucketList() (uint64, bool) {
	lcmV1, ok := l.GetV1()
	if ok {
		return uint64(lcmV1.TotalByteSizeOfBucketList), true
	}

	return 0, false
}

func (l Ledger) NodeID() (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if ok {
		nodeID, ok := utils.GetAddress(LedgerCloseValueSignature.NodeId)
		if ok {
			return nodeID, true
		}
	}

	return "", false
}

func (l Ledger) Signature() (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if ok {
		return base64.StdEncoding.EncodeToString(LedgerCloseValueSignature.Signature), true
	}

	return "", false
}

// Add docstring to larger, more complicated functions
func (l Ledger) TransactionCounts() (successTxCount, failedTxCount int32, ok bool) {
	transactions := getTransactionSet(l)
	results := l.V0.TxProcessing
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
	transactions := getTransactionSet(l)
	results := l.V0.TxProcessing
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
	switch l.V {
	case 0:
		return l.V0.TxSet.Txs
	case 1:
		switch l.V1.TxSet.V {
		case 0:
			return getTransactionPhase(l.V1.TxSet.V1TxSet.Phases)
		default:
			panic(fmt.Sprintf("unsupported LedgerCloseMeta.V1.TxSet.V: %d", l.V1.TxSet.V))
		}
	default:
		panic(fmt.Sprintf("unsupported LedgerCloseMeta.V: %d", l.V))
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
					panic(fmt.Sprintf("Unsupported TxSetComponentType: %d", component.Type))
				}

			}
		default:
			panic(fmt.Sprintf("Unsupported TransactionPhase.V: %d", phase.V))
		}
	}

	return transactionSlice
}
