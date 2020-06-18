package xdr

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLedgerSequence(t *testing.T) {
	l := LedgerCloseMeta{
		V0: &LedgerCloseMetaV0{
			LedgerHeader: LedgerHeaderHistoryEntry{
				Header: LedgerHeader{LedgerSeq: 23},
			},
		},
	}

	seq, err := l.LedgerSequence()
	assert.NoError(t, err)
	assert.Equal(t, uint32(23), seq)

	l.V = 1
	_, err = l.LedgerSequence()
	assert.EqualError(t, err, "unexpected XDR LedgerCloseMeta version: 1")
}
