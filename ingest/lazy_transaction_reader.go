package ingest

import (
	"io"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LazyTransactionReader supports reading ranges of transactions from a raw
// LedgerCloseMeta instance in a "lazy" fashion, meaning the transaction
// structures are only created when you actually request a read for that
// particular index.
type LazyTransactionReader struct {
	lcm   xdr.LedgerCloseMeta
	start int // read-only

	transactions []LedgerTransaction // cached for Rewind() calls
	lastRead     int                 // cycles through ^
}

// NewLazyTransactionReader creates a new reader instance from raw
// xdr.LedgerCloseMeta starting at a particular transaction index. Note that
// LazyTransactionReader is not thread safe and should not be shared by multiple
// goroutines.
func NewLazyTransactionReader(ledgerCloseMeta xdr.LedgerCloseMeta, start int) (*LazyTransactionReader, error) {
	if start >= ledgerCloseMeta.CountTransactions() || start < 0 {
		return nil, errors.New("'start' index exceeds ledger transaction count")
	}

	if ledgerCloseMeta.ProtocolVersion() < 20 {
		return nil, errors.New("LazyTransactionReader only works from Protocol 20 onward")
	}

	return &LazyTransactionReader{
		lcm:      ledgerCloseMeta,
		start:    start,
		lastRead: -1, // haven't started yet
	}, nil
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (reader *LazyTransactionReader) GetSequence() uint32 {
	return reader.lcm.LedgerSequence()
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (reader *LazyTransactionReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return reader.lcm.LedgerHeaderHistoryEntry()
}

// Read returns the next transaction in the ledger, ordered by tx number, each
// time it is called. When there are no more transactions to return, an EOF
// error is returned. Note that if you initialized the reader from a non-zero
// index, it will EOF when it cycles back to the start rather than when it
// reaches the end.
func (reader *LazyTransactionReader) Read() (LedgerTransaction, error) {
	txCount := reader.lcm.CountTransactions()

	i := reader.start // assume first time
	if reader.lastRead != -1 {
		i = (reader.lastRead + 1) % txCount
		if i == reader.start { // cycle, so rewind but mark as EOF
			reader.Rewind()
			return LedgerTransaction{}, io.EOF
		}
	}

	lcm := reader.lcm
	reader.lastRead = i
	envelope := lcm.TransactionEnvelopes()[i]
	cachedIdx := (i - reader.start + txCount /* to fix negatives */) % txCount

	if cachedIdx < len(reader.transactions) { // cached? return immediately
		return reader.transactions[cachedIdx], nil
	}

	newTx := LedgerTransaction{
		Index:         uint32(i + 1), // Transactions start at '1'
		Envelope:      envelope,
		Result:        lcm.TransactionResultPair(i),
		UnsafeMeta:    lcm.TxApplyProcessing(i),
		FeeChanges:    lcm.FeeProcessing(i),
		LedgerVersion: uint32(lcm.LedgerHeaderHistoryEntry().Header.LedgerVersion),
	}
	reader.transactions = append(reader.transactions, newTx)
	return newTx, nil
}

// Rewind resets the reader back to the first transaction in the ledger
func (reader *LazyTransactionReader) Rewind() {
	reader.lastRead = -1
}

// Close should be called when reading is finished. This is especially
// helpful when there are still some transactions available so reader can stop
// streaming them.
func (reader *LazyTransactionReader) Close() error {
	reader.transactions = nil
	return nil
}
