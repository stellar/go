package io

import (
	"fmt"
	"log"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type LedgerReadCloser interface {
	GetSequence() uint32
	GetHeader() xdr.LedgerHeaderHistoryEntry
	// Read should return the next transaction. If there are no more
	// transactions it should return `EOF` error.
	Read() (LedgerTransaction, error)
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some entries available so the reader can stop
	// streaming them.
	Close() error
}

type LedgerTransaction struct {
	Transaction       xdr.Transaction
	TransactionResult xdr.TransactionResult
	TransactionMeta   xdr.TransactionMeta
}

type DBLedgerReadCloser struct {
	sequence uint32
	backend  ledgerbackend.DatabaseBackend
	header   xdr.LedgerHeaderHistoryEntry
}

func (dblrc *DBLedgerReadCloser) GetSequence() uint32 {
	return dblrc.sequence
}

func (dblrc *DBLedgerReadCloser) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return dblrc.header
}

func (dblrc *DBLedgerReadCloser) Init(sequence uint32, driver string, dbURI string) error {
	dblrc.backend = ledgerbackend.DatabaseBackend{}
	err := dblrc.backend.CreateSession(driver, dbURI)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("problem instantiating backend '%s'", driver))
	}

	defer dblrc.backend.Close()

	exists, ledgerCloseMeta, err := dblrc.backend.GetLedger(sequence)

	if err != nil {
		log.Fatal("error reading ledger from backend: ", err)
	}
	if !exists {
		log.Fatalf("Ledger %d was not found", sequence)
	}

	dblrc.header = ledgerCloseMeta.LedgerHeader

	return nil
}
