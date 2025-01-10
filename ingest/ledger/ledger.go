package ledger

import (
	"encoding/base64"
	"time"

	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

type Ledger struct {
	Ledger xdr.LedgerCloseMeta
}

func (l Ledger) Sequence() uint32 {
	return uint32(l.Ledger.LedgerHeaderHistoryEntry().Header.LedgerSeq)
}

func (l Ledger) ID() int64 {
	return toid.New(int32(l.Ledger.LedgerSequence()), 0, 0).ToInt64()
}

func (l Ledger) Hash() string {
	return l.Ledger.LedgerHeaderHistoryEntry().Hash.HexString()
}

func (l Ledger) PreviousHash() string {
	return l.Ledger.PreviousLedgerHash().HexString()
}

func (l Ledger) CloseTime() int64 {
	return l.Ledger.LedgerCloseTime()
}

func (l Ledger) ClosedAt() time.Time {
	return time.Unix(l.Ledger.LedgerCloseTime(), 0).UTC()
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

func (l Ledger) SorobanFeeWrite1Kb() (int64, bool) {
	lcmV1, ok := l.Ledger.GetV1()
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
	lcmV1, ok := l.Ledger.GetV1()
	if !ok {
		return 0, false
	}

	return uint64(lcmV1.TotalByteSizeOfBucketList), true
}

func (l Ledger) NodeID() (string, bool) {
	LedgerCloseValueSignature, ok := l.Ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return "", false

	}
	return LedgerCloseValueSignature.NodeId.GetAddress()
}

func (l Ledger) Signature() (string, bool) {
	LedgerCloseValueSignature, ok := l.Ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return "", false
	}

	return base64.StdEncoding.EncodeToString(LedgerCloseValueSignature.Signature), true
}

// Add docstring to larger, more complicated functions
func (l Ledger) TransactionCounts() (successTxCount, failedTxCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := l.Ledger.TransactionEnvelopes()
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

	transactions := l.Ledger.TransactionEnvelopes()
	results = l.Ledger.TxProcessing()

	for i, result := range results {
		operations := transactions[i].Operations()
		numberOfOps := int32(len(operations))
		txSetOperationCount += numberOfOps

		// for successful transactions, the operation count is based on the operations results slice
		if result.Result.Successful() {
			operationResults, ok := result.Result.OperationResults()
			if !ok {
				return 0, 0, false
			}

			operationCount += int32(len(operationResults))
		}
	}

	return operationCount, txSetOperationCount, true
}
