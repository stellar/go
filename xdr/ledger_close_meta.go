package xdr

import "fmt"

func (l LedgerCloseMeta) LedgerHeaderHistoryEntry() LedgerHeaderHistoryEntry {
	switch l.V {
	case 0:
		return l.MustV0().LedgerHeader
	case 1:
		return l.MustV1().LedgerHeader
	case 2:
		return l.MustV2().LedgerHeader
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) LedgerSequence() uint32 {
	return uint32(l.LedgerHeaderHistoryEntry().Header.LedgerSeq)
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
	case 2:
		return len(l.MustV2().TxProcessing)
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) TransactionEnvelopes() []TransactionEnvelope {
	switch l.V {
	case 0:
		return l.MustV0().TxSet.Txs
	case 1, 2:
		var envelopes = make([]TransactionEnvelope, 0, l.CountTransactions())
		var phases []TransactionPhase
		if l.V == 1 {
			phases = l.MustV1().TxSet.V1TxSet.Phases
		} else {
			phases = l.MustV2().TxSet.V1TxSet.Phases
		}
		for _, phase := range phases {
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
	case 2:
		return l.MustV2().TxProcessing[i].Result.TransactionHash
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
	case 2:
		return l.MustV2().TxProcessing[i].Result
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
	case 2:
		return l.MustV2().TxProcessing[i].FeeProcessing
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
		return l.MustV1().TxProcessing[i].TxApplyProcessing
	case 2:
		if l.MustV2().TxProcessing[i].TxApplyProcessing.V != 3 {
			panic("TransactionResult unavailable because LedgerCloseMeta.V = 2 and TransactionMeta.V != 3")
		}
		return l.MustV2().TxProcessing[i].TxApplyProcessing
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
	case 2:
		return l.MustV2().UpgradesProcessing
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

// EvictedLedgerKeys returns the LedgerKeys for both the
// EvictedTemporaryLedgerKeys and and the EvictedPersistentLedgerEntries in a
// ledger.
func (l LedgerCloseMeta) EvictedLedgerKeys() ([]LedgerKey, error) {
	switch l.V {
	case 0, 1:
		return nil, nil
	case 2:
		v2 := l.MustV2()
		keys := make([]LedgerKey, 0, len(v2.EvictedTemporaryLedgerKeys)+len(v2.EvictedPersistentLedgerEntries))
		keys = append(keys, l.MustV2().EvictedTemporaryLedgerKeys...)
		for _, entry := range l.MustV2().EvictedPersistentLedgerEntries {
			key, err := entry.LedgerKey()
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)
		}
		return keys, nil
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}
