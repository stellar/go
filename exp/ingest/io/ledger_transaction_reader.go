package io

import (
	"io"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerTransactionReader reads transactions for a given ledger sequence from a backend.
// Use NewTransactionReader to create a new instance.
type LedgerTransactionReader struct {
	ledgerCloseMeta xdr.LedgerCloseMeta
	transactions    []LedgerTransaction
	readIdx         int
}

// NewLedgerTransactionReader creates a new TransactionReader instance.
// Note that TransactionReader is not thread safe and should not be shared by multiple goroutines
func NewLedgerTransactionReader(backend ledgerbackend.LedgerBackend, sequence uint32) (*LedgerTransactionReader, error) {
	exists, ledgerCloseMeta, err := backend.GetLedger(sequence)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ledger from the backend")
	}

	if !exists {
		return nil, ErrNotFound
	}

	reader := &LedgerTransactionReader{ledgerCloseMeta: ledgerCloseMeta}
	reader.storeTransactions(ledgerCloseMeta)
	return reader, nil
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (reader *LedgerTransactionReader) GetSequence() uint32 {
	return reader.ledgerCloseMeta.LedgerSequence()
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (reader *LedgerTransactionReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return reader.ledgerCloseMeta.V0.LedgerHeader
}

// Read returns the next transaction in the ledger, ordered by tx number, each time
// it is called. When there are no more transactions to return, an EOF error is returned.
func (reader *LedgerTransactionReader) Read() (LedgerTransaction, error) {
	if reader.readIdx < len(reader.transactions) {
		reader.readIdx++
		return reader.transactions[reader.readIdx-1], nil
	}
	return LedgerTransaction{}, io.EOF
}

// Rewind resets the reader back to the first transaction in the ledger
func (reader *LedgerTransactionReader) Rewind() {
	reader.readIdx = 0
}

// storeTransactions maps the close meta data into a slice of LedgerTransaction structs, to provide
// a per-transaction view of the data when Read() is called.
func (reader *LedgerTransactionReader) storeTransactions(lcm xdr.LedgerCloseMeta) {
	for i := range lcm.V0.TxSet.Txs {
		reader.transactions = append(reader.transactions, LedgerTransaction{
			Index:      uint32(i + 1), // Transactions start at '1'
			Envelope:   lcm.V0.TxSet.Txs[i],
			Result:     lcm.V0.TxProcessing[i].Result,
			Meta:       lcm.V0.TxProcessing[i].TxApplyProcessing,
			FeeChanges: lcm.V0.TxProcessing[i].FeeProcessing,
		})
	}
}

// Close should be called when reading is finished. This is especially
// helpful when there are still some transactions available so reader can stop
// streaming them.
func (reader *LedgerTransactionReader) Close() error {
	reader.transactions = nil
	return nil
}
