package ledgerexporter

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

// LedgerMetaArchive represents a file with metadata and binary data.
type LedgerMetaArchive struct {
	// file name
	objectKey string
	// Actual binary data
	data xdr.LedgerCloseMetaBatch
}

// NewLedgerMetaArchive creates a new LedgerMetaArchive instance.
func NewLedgerMetaArchive(key string, startSeq uint32, endSeq uint32) *LedgerMetaArchive {
	return &LedgerMetaArchive{
		objectKey: key,
		data: xdr.LedgerCloseMetaBatch{
			StartSequence: xdr.Uint32(startSeq),
			EndSequence:   xdr.Uint32(endSeq),
		},
	}
}

// AddLedger adds a LedgerCloseMeta to the archive.
func (f *LedgerMetaArchive) AddLedger(ledgerCloseMeta xdr.LedgerCloseMeta) error {
	if ledgerCloseMeta.LedgerSequence() < uint32(f.data.StartSequence) ||
		ledgerCloseMeta.LedgerSequence() > uint32(f.data.EndSequence) {
		return fmt.Errorf("ledger sequence %d is outside valid range [%d, %d]",
			ledgerCloseMeta.LedgerSequence(), f.data.StartSequence, f.data.EndSequence)
	}

	if len(f.data.LedgerCloseMetas) > 0 {
		lastSequence := f.data.LedgerCloseMetas[len(f.data.LedgerCloseMetas)-1].LedgerSequence()
		if ledgerCloseMeta.LedgerSequence() != lastSequence+1 {
			return fmt.Errorf("ledgers must be added sequentially: expected sequence %d, got %d",
				lastSequence+1, ledgerCloseMeta.LedgerSequence())
		}
	}
	f.data.LedgerCloseMetas = append(f.data.LedgerCloseMetas, ledgerCloseMeta)
	return nil
}

// GetLedgerCount returns the number of ledgers currently in the archive.
func (f *LedgerMetaArchive) GetLedgerCount() uint32 {
	return uint32(len(f.data.LedgerCloseMetas))
}

// GetStartLedgerSequence returns the starting ledger sequence of the archive.
func (f *LedgerMetaArchive) GetStartLedgerSequence() uint32 {
	return uint32(f.data.StartSequence)
}

// GetEndLedgerSequence returns the ending ledger sequence of the archive.
func (f *LedgerMetaArchive) GetEndLedgerSequence() uint32 {
	return uint32(f.data.EndSequence)
}

// GetObjectKey returns the object key of the archive.
func (f *LedgerMetaArchive) GetObjectKey() string {
	return f.objectKey
}
