package ingestadapters

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
)

// LedgerBackendAdapter provides a convenient I/O layer above the low-level LedgerBackend implementation.
type LedgerBackendAdapter struct {
	Backend *ledgerbackend.DatabaseBackend
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the backend.
func (lba *LedgerBackendAdapter) GetLatestLedgerSequence() (uint32, error) {
	err := lba.init()
	if err != nil {
		return 0, err
	}
	return lba.Backend.GetLatestLedgerSequence()
}

// GetLedger returns information about a given ledger as an object that can be streamed.
func (lba *LedgerBackendAdapter) GetLedger(sequence uint32) (io.LedgerReadCloser, error) {
	err := lba.init()
	if err != nil {
		return nil, err
	}
	dblrc := io.DBLedgerReadCloser{}
	err = dblrc.Init(sequence, lba.Backend)
	if err != nil {
		return nil, err
	}

	return &dblrc, nil
}

// Init initialises the provided backend.
func (lba *LedgerBackendAdapter) init() error {
	if lba.Backend == nil {
		return errors.New("missing LedgerBackendAdapter.Backend")
	}
	return lba.Backend.Init()
}

// Close shuts down the provided backend.
func (lba *LedgerBackendAdapter) Close() error {
	return lba.Backend.Close()
}
