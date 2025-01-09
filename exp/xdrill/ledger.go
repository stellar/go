package xdrill

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/xdr"
)

type Ledger struct {
	Ledger *xdr.LedgerCloseMeta
}

func (l Ledger) Sequence() uint32 {
	return uint32(l.Ledger.LedgerHeaderHistoryEntry().Header.LedgerSeq)
}

func (l Ledger) ID() int64 {
	return utils.NewID(int32(l.Sequence()), 0, 0).ToInt64()
}

func (l Ledger) Hash() string {
	return utils.HashToHexString(l.Ledger.LedgerHeaderHistoryEntry().Hash)
}

func (l Ledger) PreviousHash() string {
	return utils.HashToHexString(l.Ledger.PreviousLedgerHash())
}

func (l Ledger) CloseTime() int64 {
	return l.Ledger.LedgerCloseTime()
}

func (l Ledger) ClosedAt() time.Time {
	return time.Unix(l.CloseTime(), 0).UTC()
}

func (l Ledger) TotalCoins() int64 {
	return int64(l.Ledger.LedgerHeaderHistoryEntry().Header.TotalCoins)
}

func (l Ledger) FeePool() int64 {
	return int64(l.Ledger.LedgerHeaderHistoryEntry().Header.FeePool)
}

func (l Ledger) BaseFee() uint32 {
	return uint32(l.Ledger.LedgerHeaderHistoryEntry().Header.BaseFee)
}

func (l Ledger) BaseReserve() uint32 {
	return uint32(l.Ledger.LedgerHeaderHistoryEntry().Header.BaseReserve)
}

func (l Ledger) MaxTxSetSize() uint32 {
	return uint32(l.Ledger.LedgerHeaderHistoryEntry().Header.MaxTxSetSize)
}

func (l Ledger) LedgerVersion() uint32 {
	return uint32(l.Ledger.LedgerHeaderHistoryEntry().Header.LedgerVersion)
}

func (l Ledger) SorobanFeeWrite1Kb() *int64 {
	lcmV1, ok := l.Ledger.GetV1()
	if !ok {
		return nil
	}

	extV1, ok := lcmV1.Ext.GetV1()
	if !ok {
		return nil
	}

	result := int64(extV1.SorobanFeeWrite1Kb)

	return &result
}

func (l Ledger) TotalByteSizeOfBucketList() *uint64 {
	lcmV1, ok := l.Ledger.GetV1()
	if !ok {
		return nil
	}

	result := uint64(lcmV1.TotalByteSizeOfBucketList)

	return &result
}

func (l Ledger) NodeID() *string {
	LedgerCloseValueSignature, ok := l.Ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return nil

	}
	nodeID, ok := utils.GetAddress(LedgerCloseValueSignature.NodeId)
	if !ok {
		return nil
	}

	return &nodeID
}

func (l Ledger) Signature() *string {
	LedgerCloseValueSignature, ok := l.Ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return nil
	}

	result := base64.StdEncoding.EncodeToString(LedgerCloseValueSignature.Signature)

	return &result
}

// Add docstring to larger, more complicated functions
func (l Ledger) TransactionCounts() (successTxCount, failedTxCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := getTransactionSet(l)
	results = l.Ledger.TxProcessing()
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
	results = l.Ledger.TxProcessing()

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
	switch l.Ledger.V {
	case 0:
		return l.Ledger.V0.TxSet.Txs
	case 1:
		switch l.Ledger.V1.TxSet.V {
		case 0:
			return getTransactionPhase(l.Ledger.V1.TxSet.V1TxSet.Phases)
		default:
			panic(fmt.Sprintf("unsupported LedgerCloseMeta.V1.TxSet.V: %d", l.Ledger.V1.TxSet.V))
		}
	default:
		panic(fmt.Sprintf("unsupported LedgerCloseMeta.V: %d", l.Ledger.V))
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
