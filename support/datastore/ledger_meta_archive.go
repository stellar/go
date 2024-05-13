package datastore

import (
	"fmt"

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

// AddLedger adds a LedgerCloseMeta to the archive.
func (f *LedgerMetaArchive) AddLedger(ledgerCloseMeta xdr.LedgerCloseMeta) error {
	if ledgerCloseMeta.LedgerSequence() < uint32(f.Data.StartSequence) ||
		ledgerCloseMeta.LedgerSequence() > uint32(f.Data.EndSequence) {
		return fmt.Errorf("ledger sequence %d is outside valid range [%d, %d]",
			ledgerCloseMeta.LedgerSequence(), f.Data.StartSequence, f.Data.EndSequence)
	}

	if len(f.Data.LedgerCloseMetas) > 0 {
		lastSequence := f.Data.LedgerCloseMetas[len(f.Data.LedgerCloseMetas)-1].LedgerSequence()
		if ledgerCloseMeta.LedgerSequence() != lastSequence+1 {
			return fmt.Errorf("ledgers must be added sequentially: expected sequence %d, got %d",
				lastSequence+1, ledgerCloseMeta.LedgerSequence())
		}
	}
	f.Data.LedgerCloseMetas = append(f.Data.LedgerCloseMetas, ledgerCloseMeta)
	return nil
}

// GetLedger retrieves the LedgerCloseMeta for a given sequence number.
// It returns an error if the sequence number is invalid or if the LedgerCloseMeta for the sequence number is not found.
func (f *LedgerMetaArchive) GetLedger(sequence uint32) (xdr.LedgerCloseMeta, error) {
	if sequence < uint32(f.Data.StartSequence) || sequence > uint32(f.Data.EndSequence) {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("ledger sequence %d is outside the "+
			"valid range of ledger sequences [%d, %d] this meta archive holds",
			sequence, f.Data.StartSequence, f.Data.EndSequence)
	}

	ledgerIndex := sequence - uint32(f.Data.StartSequence)
	if ledgerIndex >= uint32(len(f.Data.LedgerCloseMetas)) {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("LedgerCloseMeta for sequence %d is "+
			"not found in %s meta archive", sequence, f.ObjectKey)
	}
	return f.Data.LedgerCloseMetas[ledgerIndex], nil
}
