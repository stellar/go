package ingestadapters

import (
	"errors"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
)

const noBackendErr = "missing LedgerBackendAdapter.Backend"

// LedgerBackendAdapter provides a convenient I/O layer above the low-level LedgerBackend implementation.
type LedgerBackendAdapter struct {
	Backend ledgerbackend.LedgerBackend
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the backend.
func (lba *LedgerBackendAdapter) GetLatestLedgerSequence() (uint32, error) {
	if !lba.hasBackend() {
		return 0, errors.New(noBackendErr)
	}
	return lba.Backend.GetLatestLedgerSequence()
}

// GetLedger returns information about a given ledger as an object that can be streamed.
func (lba *LedgerBackendAdapter) GetLedger(sequence uint32) (io.LedgerReadCloser, error) {
	if !lba.hasBackend() {
		return nil, errors.New(noBackendErr)
	}

	dblrc := io.MakeLedgerReadCloser(sequence, lba.Backend)
	return dblrc, nil
}

// Close shuts down the provided backend.
func (lba *LedgerBackendAdapter) Close() error {
	if !lba.hasBackend() {
		return errors.New(noBackendErr)
	}
	return lba.Backend.Close()
}

// hasBackend checks for the presence of LedgerBackendAdapter.Backend.
func (lba *LedgerBackendAdapter) hasBackend() bool {
	return lba.Backend != nil
}
