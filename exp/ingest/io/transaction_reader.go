package io

import (
	"io"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
)

// TransactionReader reads transactions for a given ledger sequence from a backend.
// Use NewTransactionReader to create a new instance.
type TransactionReader struct {
	ledgerReader *LedgerReader
	transactions []LedgerTransaction
	readIdx      int
}

// NewTransactionReader creates a new TransactionReader instance.
// Note that TransactionReader is not thread safe and should not be shared by multiple goroutines
func NewTransactionReader(backend ledgerbackend.LedgerBackend, sequence uint32) (*TransactionReader, error) {
	ledgerReader, err := NewLedgerReader(backend, sequence)
	if err != nil {
		return nil, err
	}

	reader := &TransactionReader{ledgerReader: ledgerReader}
	reader.storeTransactions(ledgerReader.ledgerCloseMeta)
	return reader, nil
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (reader *TransactionReader) GetSequence() uint32 {
	return reader.ledgerReader.GetSequence()
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (reader *TransactionReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return reader.ledgerReader.GetHeader()
}

// Read returns the next transaction in the ledger, ordered by tx number, each time
// it is called. When there are no more transactions to return, an EOF error is returned.
func (reader *TransactionReader) Read() (LedgerTransaction, error) {
	if reader.readIdx < len(reader.transactions) {
		reader.readIdx++
		return reader.transactions[reader.readIdx-1], nil
	}
	return LedgerTransaction{}, io.EOF
}

// Rewind resets the reader back to the first transaction in the ledger
func (reader *TransactionReader) Rewind() {
	reader.readIdx = 0
}

// storeTransactions maps the close meta data into a slice of LedgerTransaction structs, to provide
// a per-transaction view of the data when Read() is called.
func (reader *TransactionReader) storeTransactions(lcm ledgerbackend.LedgerCloseMeta) {
	for i := range lcm.TransactionEnvelope {
		reader.transactions = append(reader.transactions, LedgerTransaction{
			Index:      uint32(i + 1), // Transactions start at '1'
			Envelope:   lcm.TransactionEnvelope[i],
			Result:     lcm.TransactionResult[i],
			Meta:       lcm.TransactionMeta[i],
			FeeChanges: lcm.TransactionFeeChanges[i],
		})
	}
}

// Close should be called when reading is finished. This is especially
// helpful when there are still some transactions available so reader can stop
// streaming them.
func (reader *TransactionReader) Close() error {
	reader.transactions = nil
	return nil
}
