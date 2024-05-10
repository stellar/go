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

// GetLedgerCount returns the number of ledgers currently in the archive.
func (f *LedgerMetaArchive) GetLedgerCount() uint32 {
	return uint32(len(f.Data.LedgerCloseMetas))
}

// GetStartLedgerSequence returns the starting ledger sequence of the archive.
func (f *LedgerMetaArchive) GetStartLedgerSequence() uint32 {
	return uint32(f.Data.StartSequence)
}

// GetEndLedgerSequence returns the ending ledger sequence of the archive.
func (f *LedgerMetaArchive) GetEndLedgerSequence() uint32 {
	return uint32(f.Data.EndSequence)
}

// GetObjectKey returns the object key of the archive.
func (f *LedgerMetaArchive) GetObjectKey() string {
	return f.ObjectKey
}

func (f *LedgerMetaArchive) GetLedger(sequence uint32) (xdr.LedgerCloseMeta, error) {
	if sequence < uint32(f.Data.StartSequence) || sequence > uint32(f.Data.EndSequence) {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("ledger sequence %d is outside valid range [%d, %d]",
			sequence, f.Data.StartSequence, f.Data.EndSequence)
	}

	ledgerIndex := sequence - f.GetStartLedgerSequence()
	if ledgerIndex >= uint32(len(f.Data.LedgerCloseMetas)) {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("LedgerCloseMeta for sequence %d not found", sequence)
	}
	return f.Data.LedgerCloseMetas[ledgerIndex], nil
}

func CreateLedgerCloseMeta(ledgerSeq uint32) xdr.LedgerCloseMeta {
	return xdr.LedgerCloseMeta{
		V: int32(0),
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(ledgerSeq),
				},
			},
			TxSet:              xdr.TransactionSet{},
			TxProcessing:       nil,
			UpgradesProcessing: nil,
			ScpInfo:            nil,
		},
		V1: nil,
	}
}
