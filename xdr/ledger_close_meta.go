package xdr

import (
	"encoding/base64"
	"fmt"
	"time"
)

func (l LedgerCloseMeta) LedgerHeaderHistoryEntry() LedgerHeaderHistoryEntry {
	switch l.V {
	case 0:
		return l.MustV0().LedgerHeader
	case 1:
		return l.MustV1().LedgerHeader
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) LedgerSequence() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.LedgerSeq)
}

func (l LedgerCloseMeta) LedgerCloseTime() int64 {
	return int64(l.LedgerHeaderHistoryEntry().Header.ScpValue.CloseTime)
}

func (l LedgerCloseMeta) LedgerHash() Hash {
	return l.LedgerHeaderHistoryEntry().Hash
}

func (l LedgerCloseMeta) PreviousLedgerHash() Hash {
	return l.LedgerHeaderHistoryEntry().Header.PreviousLedgerHash
}

func (l LedgerCloseMeta) ProtocolVersion() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.LedgerVersion)
}

func (l LedgerCloseMeta) BucketListHash() Hash {
	return l.LedgerHeaderHistoryEntry().Header.BucketListHash
}

func (l LedgerCloseMeta) CountTransactions() int {
	switch l.V {
	case 0:
		return len(l.MustV0().TxProcessing)
	case 1:
		return len(l.MustV1().TxProcessing)

	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) TransactionEnvelopes() []TransactionEnvelope {
	switch l.V {
	case 0:
		return l.MustV0().TxSet.Txs
	case 1:
		var envelopes = make([]TransactionEnvelope, 0, l.CountTransactions())
		for _, phase := range l.MustV1().TxSet.V1TxSet.Phases {
			for _, component := range *phase.V0Components {
				envelopes = append(envelopes, component.TxsMaybeDiscountedFee.Txs...)
			}
		}
		return envelopes
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// TransactionHash returns Hash for tx at index i in processing order..
func (l LedgerCloseMeta) TransactionHash(i int) Hash {
	switch l.V {
	case 0:
		return l.MustV0().TxProcessing[i].Result.TransactionHash
	case 1:
		return l.MustV1().TxProcessing[i].Result.TransactionHash
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// TransactionResultPair returns TransactionResultPair for tx at index i in processing order.
func (l LedgerCloseMeta) TransactionResultPair(i int) TransactionResultPair {
	switch l.V {
	case 0:
		return l.MustV0().TxProcessing[i].Result
	case 1:
		return l.MustV1().TxProcessing[i].Result
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// FeeProcessing returns FeeProcessing for tx at index i in processing order.
func (l LedgerCloseMeta) FeeProcessing(i int) LedgerEntryChanges {
	switch l.V {
	case 0:
		return l.MustV0().TxProcessing[i].FeeProcessing
	case 1:
		return l.MustV1().TxProcessing[i].FeeProcessing
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// TxApplyProcessing returns TxApplyProcessing for tx at index i in processing order.
func (l LedgerCloseMeta) TxApplyProcessing(i int) TransactionMeta {
	switch l.V {
	case 0:
		return l.MustV0().TxProcessing[i].TxApplyProcessing
	case 1:
		if l.MustV1().TxProcessing[i].TxApplyProcessing.V != 3 {
			panic("TransactionResult unavailable because LedgerCloseMeta.V = 1 and TransactionMeta.V != 3")
		}
		return l.MustV1().TxProcessing[i].TxApplyProcessing
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// UpgradesProcessing returns UpgradesProcessing for ledger.
func (l LedgerCloseMeta) UpgradesProcessing() []UpgradeEntryMeta {
	switch l.V {
	case 0:
		return l.MustV0().UpgradesProcessing
	case 1:
		return l.MustV1().UpgradesProcessing
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// EvictedTemporaryLedgerKeys returns a slice of ledger keys for
// temporary ledger entries that have been evicted in this ledger.
func (l LedgerCloseMeta) EvictedTemporaryLedgerKeys() ([]LedgerKey, error) {
	switch l.V {
	case 0:
		return nil, nil
	case 1:
		return l.MustV1().EvictedTemporaryLedgerKeys, nil
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// EvictedPersistentLedgerEntries returns the persistent ledger entries
// which have been evicted in this ledger.
func (l LedgerCloseMeta) EvictedPersistentLedgerEntries() ([]LedgerEntry, error) {
	switch l.V {
	case 0:
		return nil, nil
	case 1:
		return l.MustV1().EvictedPersistentLedgerEntries, nil
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) LedgerID() int64 {
	return NewID(int32(l.LedgerSequence()), 0, 0).ToInt64()
}

func (l LedgerCloseMeta) LedgerClosedAt() time.Time {
	return time.Unix(l.LedgerCloseTime(), 0).UTC()
}

func (l LedgerCloseMeta) TotalCoins() int64 {
	return int64(l.LedgerHeaderHistoryEntry().Header.TotalCoins)
}

func (l LedgerCloseMeta) FeePool() int64 {
	return int64(l.LedgerHeaderHistoryEntry().Header.FeePool)
}

func (l LedgerCloseMeta) BaseFee() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.BaseFee)
}

func (l LedgerCloseMeta) BaseReserve() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.BaseReserve)
}

func (l LedgerCloseMeta) MaxTxSetSize() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.MaxTxSetSize)
}

func (l LedgerCloseMeta) SorobanFeeWrite1Kb() (int64, bool) {
	lcmV1, ok := l.GetV1()
	if ok {
		extV1 := lcmV1.Ext.MustV1()
		return int64(extV1.SorobanFeeWrite1Kb), true
	}

	return 0, false
}

func (l LedgerCloseMeta) TotalByteSizeOfBucketList() (uint64, bool) {
	lcmV1, ok := l.GetV1()
	if ok {
		return uint64(lcmV1.TotalByteSizeOfBucketList), true
	}

	return 0, false
}

func (l LedgerCloseMeta) NodeID() (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if ok {
		nodeID, ok := GetAddress(LedgerCloseValueSignature.NodeId)
		if ok {
			return nodeID, true
		}
	}

	return "", false
}

func (l LedgerCloseMeta) Signature() (string, bool) {
	LedgerCloseValueSignature, ok := l.LedgerHeaderHistoryEntry().Header.ScpValue.Ext.GetLcValueSignature()
	if ok {
		return base64.StdEncoding.EncodeToString(LedgerCloseValueSignature.Signature), true
	}

	return "", false
}

func (l LedgerCloseMeta) TransactionProcessing() []TransactionResultMeta {
	switch l.V {
	case 0:
		return l.MustV0().TxProcessing
	case 1:
		return l.MustV1().TxProcessing

	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) CountSuccessfulTransactions() int {
	var successfulTransactionCount int
	results := l.TransactionProcessing()

	for _, result := range results {
		if result.Result.Successful() {
			successfulTransactionCount++
		}
	}

	return successfulTransactionCount
}

func (l LedgerCloseMeta) CountFailedTransactions() int {
	var failedTransactionCount int
	results := l.TransactionProcessing()

	for _, result := range results {
		if !result.Result.Successful() {
			failedTransactionCount++
		}
	}

	return failedTransactionCount
}

func (l LedgerCloseMeta) CountOperations() (operationCount int) {
	transactions := l.TransactionEnvelopes()

	for _, transaction := range transactions {
		operations := transaction.Operations()
		numberOfOps := len(operations)
		operationCount += numberOfOps
	}

	return
}

func (l LedgerCloseMeta) CountSuccessfulOperations() (operationCount int) {
	results := l.TransactionProcessing()

	for _, result := range results {
		if result.Result.Successful() {
			operationResults, ok := result.Result.OperationResults()
			if !ok {
				panic("could not get []OperationResult")
			}

			operationCount += len(operationResults)
		}
	}

	return
}

func (l LedgerCloseMeta) CountFailedOperations() (operationCount int) {
	results := l.TransactionProcessing()

	for _, result := range results {
		if !result.Result.Successful() {
			operationResults, ok := result.Result.OperationResults()
			if !ok {
				panic("could not get []OperationResult")
			}

			operationCount += len(operationResults)
		}
	}

	return
}
