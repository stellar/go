package xdr

import "fmt"

func (l LedgerCloseMeta) LedgerSequence() uint32 {
	switch l.V {
	case 0:
		return uint32(l.MustV0().LedgerHeader.Header.LedgerSeq)
	case 1:
		return uint32(l.MustV1().LedgerHeader.Header.LedgerSeq)
	case 2:
		return uint32(l.MustV2().LedgerHeader.Header.LedgerSeq)
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) LedgerHash() Hash {
	switch l.V {
	case 0:
		return l.MustV0().LedgerHeader.Hash
	case 1:
		return l.MustV1().LedgerHeader.Hash
	case 2:
		return l.MustV2().LedgerHeader.Hash
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) PreviousLedgerHash() Hash {
	switch l.V {
	case 0:
		return l.MustV0().LedgerHeader.Header.PreviousLedgerHash
	case 1:
		return l.MustV1().LedgerHeader.Header.PreviousLedgerHash
	case 2:
		return l.MustV2().LedgerHeader.Header.PreviousLedgerHash
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) ProtocolVersion() uint32 {
	switch l.V {
	case 0:
		return uint32(l.MustV0().LedgerHeader.Header.LedgerVersion)
	case 1:
		return uint32(l.MustV1().LedgerHeader.Header.LedgerVersion)
	case 2:
		return uint32(l.MustV2().LedgerHeader.Header.LedgerVersion)
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}

func (l LedgerCloseMeta) BucketListHash() Hash {
	switch l.V {
	case 0:
		return l.MustV0().LedgerHeader.Header.BucketListHash
	case 1:
		return l.MustV1().LedgerHeader.Header.BucketListHash
	case 2:
		return l.MustV2().LedgerHeader.Header.BucketListHash
	default:
		panic(fmt.Sprintf("Unsupported LedgerCloseMeta.V: %d", l.V))
	}
}
