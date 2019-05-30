package ingestadapters

import (
	"fmt"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
)

// LedgerBackendAdapter provides a convenient I/O layer above the low-level LedgerBackend implementation.
type LedgerBackendAdapter struct {
	backend ledgerbackend.LedgerBackend
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the backend.
func (lba *LedgerBackendAdapter) GetLatestLedgerSequence() (uint32, error) {
	return lba.backend.GetLatestLedgerSequence()
}

// GetLedger returns...
func (lba *LedgerBackendAdapter) GetLedger(uint32) (io.LedgerReader, error) {
	// func (lba *LedgerBackendAdapter) GetLedger(uint32) (io.LedgerReadCloser, error) {
	// TODO: placeholder
	return nil, fmt.Errorf("not implemented yet")
}
