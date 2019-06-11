package ingestadapters

import (
	"fmt"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
)

// LedgerBackendAdapter provides a convenient I/O layer above the low-level LedgerBackend implementation.
type LedgerBackendAdapter struct {
	backend *ledgerbackend.DatabaseBackend
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the backend.
func (lba *LedgerBackendAdapter) GetLatestLedgerSequence() (uint32, error) {
	return lba.backend.GetLatestLedgerSequence()
}

// GetLedger returns information about a given ledger as an object that can be streamed.
func (lba *LedgerBackendAdapter) GetLedger(sequence uint32) (io.LedgerReadCloser, error) {
	// TODO: Would like to only initialise session once, not on every GetLedger
	// TODO: Don't init unless driver and dburi are set
	dblrc := io.DBLedgerReadCloser{}
	err := dblrc.Init(sequence, lba.backend)
	if err != nil {
		return nil, err
	}

	return &dblrc, nil
}

func (lba *LedgerBackendAdapter) Init(driver string, dbURI string) error {
	lba.backend = &ledgerbackend.DatabaseBackend{}
	err := lba.backend.CreateSession(driver, dbURI)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("problem instantiating backend '%s'", driver))
	}

	return nil
}

func (lba *LedgerBackendAdapter) Close() error {
	return lba.backend.Close()
}
