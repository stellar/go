package xdr

import "fmt"

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
	return uint32(l.MustV0().LedgerHeader.Header.LedgerSeq)
}

func (l LedgerCloseMeta) LedgerHash() Hash {
	return l.MustV0().LedgerHeader.Hash
}

func (l LedgerCloseMeta) PreviousLedgerHash() Hash {
	return l.MustV0().LedgerHeader.Header.PreviousLedgerHash
}

func (l LedgerCloseMeta) ProtocolVersion() uint32 {
	return uint32(l.MustV0().LedgerHeader.Header.LedgerVersion)
}

func (l LedgerCloseMeta) BucketListHash() Hash {
	return l.MustV0().LedgerHeader.Header.BucketListHash
}
