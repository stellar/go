package ingestadapters

import (
	"errors"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
)

// LedgerBackendAdapter provides a convenient I/O layer above the low-level LedgerBackend implementation.
type LedgerBackendAdapter struct {
	Backend *ledgerbackend.DatabaseBackend
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the backend.
func (lba *LedgerBackendAdapter) GetLatestLedgerSequence() (uint32, error) {
	if !lba.hasBackend() {
		return 0, errors.New("missing LedgerBackendAdapter.Backend")
	}
	return lba.Backend.GetLatestLedgerSequence()
}

// GetLedger returns information about a given ledger as an object that can be streamed.
func (lba *LedgerBackendAdapter) GetLedger(sequence uint32) (io.LedgerReadCloser, error) {
	if !lba.hasBackend() {
		return nil, errors.New("missing LedgerBackendAdapter.Backend")
	}

	dblrc := io.DBLedgerReadCloser{}
	err := dblrc.Init(sequence, lba.Backend)
	if err != nil {
		return nil, err
	}

	return &dblrc, nil
}

// hasBackend
func (lba *LedgerBackendAdapter) hasBackend() bool {
	if lba.Backend == nil {
		return false
	}

	return true
}

// Close shuts down the provided backend.
func (lba *LedgerBackendAdapter) Close() error {
	return lba.Backend.Close()
}
