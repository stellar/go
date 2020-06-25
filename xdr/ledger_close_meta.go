package xdr

func (l LedgerCloseMeta) LedgerSequence() uint32 {
	return uint32(l.MustV0().LedgerHeader.Header.LedgerSeq)
}
