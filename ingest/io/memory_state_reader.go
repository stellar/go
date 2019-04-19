package io

import (
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// MemoryStateReader is the in-memory representation of HistoryArchiveState
type MemoryStateReader struct {
	has *historyarchive.HistoryArchiveState
}

// MakeMemoryStateReader is a factory method for MemoryStateReader
func MakeMemoryStateReader(has *historyarchive.HistoryArchiveState) *MemoryStateReader {
	return &MemoryStateReader{has: has}
}

// enforce MemoryStateReader to implement StateReader
var _ StateReader = &MemoryStateReader{}

// GetSequence placeholder
func (msr *MemoryStateReader) GetSequence() uint32 {
	return msr.has.CurrentLedger
}

// Read placeholder
func (msr *MemoryStateReader) Read() (bool, xdr.LedgerEntry, error) {
	return true, xdr.LedgerEntry{}, nil
}
