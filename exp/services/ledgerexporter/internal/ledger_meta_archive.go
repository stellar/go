package ledgerexporter

import (
	"github.com/stellar/go/xdr"
)

// LedgerMetaArchive represents a file with metadata and binary data.
type LedgerMetaArchive struct {
	// file name
	ObjectKey string
	// Actual binary data
	Data xdr.LedgerCloseMetaBatch
}

// NewLedgerMetaArchive creates a new LedgerMetaArchive instance.
func NewLedgerMetaArchive(key string, startSeq uint32, endSeq uint32) *LedgerMetaArchive {
	return &LedgerMetaArchive{
		ObjectKey: key,
		Data: xdr.LedgerCloseMetaBatch{
			StartSequence: xdr.Uint32(startSeq),
			EndSequence:   xdr.Uint32(endSeq),
		},
	}
}
