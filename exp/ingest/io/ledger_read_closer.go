package io

import (
	"io"
	"log"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
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
	Index    uint32
	Envelope xdr.TransactionEnvelope
	Result   xdr.TransactionResultPair
	Meta     xdr.TransactionMeta
}

type DBLedgerReadCloser struct {
	sequence     uint32
	header       xdr.LedgerHeaderHistoryEntry
	transactions []LedgerTransaction
	readIdx      int
}

// Ensure DatabaseBackend implements LedgerBackend
var _ LedgerReadCloser = (*DBLedgerReadCloser)(nil)

func (dblrc *DBLedgerReadCloser) GetSequence() uint32 {
	return dblrc.sequence
}

func (dblrc *DBLedgerReadCloser) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return dblrc.header
}

func (dblrc *DBLedgerReadCloser) Read() (LedgerTransaction, error) {
	if dblrc.readIdx < len(dblrc.transactions) {
		defer dblrc.incReadIdx()
		return dblrc.transactions[dblrc.readIdx], nil
	}
	return LedgerTransaction{}, io.EOF
}

func (dblrc *DBLedgerReadCloser) Close() error {
	// TODO - raise an error if no data initialised yet
	dblrc.readIdx = len(dblrc.transactions)
	return nil
}

func (dblrc *DBLedgerReadCloser) Init(sequence uint32, backend *ledgerbackend.DatabaseBackend) error {
	exists, ledgerCloseMeta, err := backend.GetLedger(sequence)

	if err != nil {
		log.Fatal("error reading ledger from backend: ", err)
	}
	if !exists {
		log.Fatalf("Ledger %d was not found", sequence)
	}

	dblrc.sequence = sequence
	dblrc.header = ledgerCloseMeta.LedgerHeader

	dblrc.storeTransactions(ledgerCloseMeta)

	return nil
}

func (dblrc *DBLedgerReadCloser) storeTransactions(lcm ledgerbackend.LedgerCloseMeta) {
	// TODO: Assume all slices are the same length - do we need to verify that?
	// TODO: This should only be done once - how to enforce?
	for i := range lcm.TransactionEnvelope {
		dblrc.transactions = append(dblrc.transactions, LedgerTransaction{
			Index:    lcm.TransactionIndex[i],
			Envelope: lcm.TransactionEnvelope[i],
			Result:   lcm.TransactionResult[i],
			Meta:     lcm.TransactionMeta[i],
		})
	}
}

func (dblrc *DBLedgerReadCloser) incReadIdx() {
	dblrc.readIdx++
}
