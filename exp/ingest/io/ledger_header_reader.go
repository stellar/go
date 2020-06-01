package io

import (
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerReader reads ledger header from a given backend and ledger sequence.
// Use NewLedgerReader to create a new instance.
type LedgerReader struct {
	ledgerCloseMeta ledgerbackend.LedgerCloseMeta
}

// NewLedgerReader creates a new LedgerReader instance.
func NewLedgerReader(backend ledgerbackend.LedgerBackend, sequence uint32) (*LedgerReader, error) {
	exists, ledgerCloseMeta, err := backend.GetLedger(sequence)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ledger from the backend")
	}

	if !exists {
		return nil, ErrNotFound
	}

	return &LedgerReader{
		ledgerCloseMeta: ledgerCloseMeta,
	}, nil
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (reader *LedgerReader) GetSequence() uint32 {
	return uint32(reader.ledgerCloseMeta.LedgerHeader.Header.LedgerSeq)
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (reader *LedgerReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return reader.ledgerCloseMeta.LedgerHeader
}
