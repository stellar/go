package ledger

import (
	"encoding/base64"
	"time"

	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

func Sequence(l xdr.LedgerCloseMeta) uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.LedgerSeq)
}

func ID(l xdr.LedgerCloseMeta) int64 {
	return toid.New(int32(l.LedgerSequence()), 0, 0).ToInt64()
}

func Hash(l xdr.LedgerCloseMeta) string {
	return l.LedgerHeaderHistoryEntry().Hash.HexString()
}

func PreviousHash(l xdr.LedgerCloseMeta) string {
	return l.PreviousLedgerHash().HexString()
}

func CloseTime(l xdr.LedgerCloseMeta) int64 {
	return l.LedgerCloseTime()
}

func ClosedAt(l xdr.LedgerCloseMeta) time.Time {
	return time.Unix(l.LedgerCloseTime(), 0).UTC()
}

func TotalCoins(l xdr.LedgerCloseMeta) int64 {
	return int64(l.LedgerHeaderHistoryEntry().Header.TotalCoins)
}

func FeePool(l xdr.LedgerCloseMeta) int64 {
	return int64(l.LedgerHeaderHistoryEntry().Header.FeePool)
}

func BaseFee(l xdr.LedgerCloseMeta) uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.BaseFee)
}

func BaseReserve(l xdr.LedgerCloseMeta) uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.BaseReserve)
}

func MaxTxSetSize(l xdr.LedgerCloseMeta) uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.MaxTxSetSize)
}

func LedgerVersion(l xdr.LedgerCloseMeta) uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.LedgerVersion)
}

func SorobanFeeWrite1Kb(l xdr.LedgerCloseMeta) (int64, bool) {
	lcmV1, ok := l.GetV1()
	if !ok {
		return 0, false
	}

	extV1, ok := lcmV1.Ext.GetV1()
	if !ok {
		return 0, false
	}

	return int64(extV1.SorobanFeeWrite1Kb), true
}

func TotalByteSizeOfBucketList(l xdr.LedgerCloseMeta) (uint64, bool) {
	lcmV1, ok := l.GetV1()
	if !ok {
		return 0, false
	}

	return uint64(lcmV1.TotalByteSizeOfBucketList), true
}

func NodeID(l xdr.LedgerCloseMeta) (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return "", false

	}
	return LedgerCloseValueSignature.NodeId.GetAddress()
}

func Signature(l xdr.LedgerCloseMeta) (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if !ok {
		return "", false
	}

	return base64.StdEncoding.EncodeToString(LedgerCloseValueSignature.Signature), true
}

// Add docstring to larger, more complicated functions
func TransactionCounts(l xdr.LedgerCloseMeta) (successTxCount, failedTxCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := l.TransactionEnvelopes()
	results = l.TxProcessing()
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
func OperationCounts(l xdr.LedgerCloseMeta) (operationCount, txSetOperationCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := l.TransactionEnvelopes()
	results = l.TxProcessing()

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
