package xdr

import "fmt"

func (l LedgerCloseMeta) LedgerSequence() (uint32, error) {
	v0, ok := l.GetV0()
	if !ok {
		return 0, fmt.Errorf("unexpected XDR LedgerCloseMeta version: %v", l.V)
	}
	return uint32(v0.LedgerHeader.Header.LedgerSeq), nil
}
