package io

import (
	"fmt"

	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// MemoryStateReader is the in-memory representation of HistoryArchiveState
type MemoryStateReader struct {
	has      *historyarchive.HistoryArchiveState
	archive  *historyarchive.Archive
	sequence uint32
}

// enforce MemoryStateReader to implement StateReader
var _ StateReader = &MemoryStateReader{}

// MakeMemoryStateReader is a factory method for MemoryStateReader
func MakeMemoryStateReader(archive *historyarchive.Archive, sequence uint32) (*MemoryStateReader, error) {
	has, e := archive.GetCheckpointHAS(sequence)
	if e != nil {
		return nil, fmt.Errorf("unable to get checkpoint HAS at ledger sequence %d: %s", sequence, e)
	}

	return &MemoryStateReader{
		has:      &has,
		archive:  archive,
		sequence: sequence,
	}, nil
}

// GetSequence placeholder
func (msr *MemoryStateReader) GetSequence() uint32 {
	return msr.sequence
}

// Read placeholder
func (msr *MemoryStateReader) Read() (bool, xdr.LedgerEntry, error) {
	for _, hash := range msr.has.Buckets() {
		if !msr.archive.BucketExists(hash) {
			return false, xdr.LedgerEntry{}, fmt.Errorf("bucket hash does not exist: %s", hash)
		}

		e := msr.archive.VerifyBucketEntries(hash)
		if e != nil {
			return false, xdr.LedgerEntry{}, fmt.Errorf("unable to verify bucket entries for hash (%s): %s", hash, e)
		}

		// msr.archive.GetXdrStream
	}

	return true, xdr.LedgerEntry{}, nil
}
