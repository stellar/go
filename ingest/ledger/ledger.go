package ledger

import (
	"encoding/base64"
	"time"

	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> origin/5550/xdrill-transactions
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
<<<<<<< HEAD
=======
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
>>>>>>> b5feb2c8 (Move xdrill ledger functions to ingest as subpackage)
=======
>>>>>>> origin/5550/xdrill-transactions
	if !ok {
		return 0, false
	}

	extV1, ok := lcmV1.Ext.GetV1()
	if !ok {
		return 0, false
	}

	return int64(extV1.SorobanFeeWrite1Kb), true
}

<<<<<<< HEAD
<<<<<<< HEAD
func TotalByteSizeOfBucketList(l xdr.LedgerCloseMeta) (uint64, bool) {
	lcmV1, ok := l.GetV1()
=======
func (l Ledger) TotalByteSizeOfBucketList() (uint64, bool) {
	lcmV1, ok := l.Ledger.GetV1()
>>>>>>> b5feb2c8 (Move xdrill ledger functions to ingest as subpackage)
=======
func TotalByteSizeOfBucketList(l xdr.LedgerCloseMeta) (uint64, bool) {
	lcmV1, ok := l.GetV1()
>>>>>>> origin/5550/xdrill-transactions
	if !ok {
		return 0, false
	}

	return uint64(lcmV1.TotalByteSizeOfBucketList), true
}

<<<<<<< HEAD
<<<<<<< HEAD
func NodeID(l xdr.LedgerCloseMeta) (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
=======
func (l Ledger) NodeID() (string, bool) {
	LedgerCloseValueSignature, ok := l.Ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
>>>>>>> b5feb2c8 (Move xdrill ledger functions to ingest as subpackage)
=======
func NodeID(l xdr.LedgerCloseMeta) (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
>>>>>>> origin/5550/xdrill-transactions
	if !ok {
		return "", false

	}
	return LedgerCloseValueSignature.NodeId.GetAddress()
}

<<<<<<< HEAD
<<<<<<< HEAD
func Signature(l xdr.LedgerCloseMeta) (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
=======
func (l Ledger) Signature() (string, bool) {
	LedgerCloseValueSignature, ok := l.Ledger.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
>>>>>>> b5feb2c8 (Move xdrill ledger functions to ingest as subpackage)
=======
func Signature(l xdr.LedgerCloseMeta) (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
>>>>>>> origin/5550/xdrill-transactions
	if !ok {
		return "", false
	}

	return base64.StdEncoding.EncodeToString(LedgerCloseValueSignature.Signature), true
}

// Add docstring to larger, more complicated functions
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> origin/5550/xdrill-transactions
func TransactionCounts(l xdr.LedgerCloseMeta) (successTxCount, failedTxCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := l.TransactionEnvelopes()
	results = l.TxProcessing()
<<<<<<< HEAD
=======
func (l Ledger) TransactionCounts() (successTxCount, failedTxCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := l.Ledger.TransactionEnvelopes()
	results = l.Ledger.TxProcessing()
>>>>>>> b5feb2c8 (Move xdrill ledger functions to ingest as subpackage)
=======
>>>>>>> origin/5550/xdrill-transactions
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
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> origin/5550/xdrill-transactions
func OperationCounts(l xdr.LedgerCloseMeta) (operationCount, txSetOperationCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := l.TransactionEnvelopes()
	results = l.TxProcessing()
<<<<<<< HEAD
=======
func (l Ledger) OperationCounts() (operationCount, txSetOperationCount int32, ok bool) {
	var results []xdr.TransactionResultMeta

	transactions := l.Ledger.TransactionEnvelopes()
	results = l.Ledger.TxProcessing()
>>>>>>> b5feb2c8 (Move xdrill ledger functions to ingest as subpackage)
=======
>>>>>>> origin/5550/xdrill-transactions

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
