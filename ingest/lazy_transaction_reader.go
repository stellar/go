package ingest

import (
	"io"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LazyTransactionReader supports reading ranges of transactions from a raw
// LedgerCloseMeta instance in a "lazy" fashion, meaning the transaction
// structures are only created when you actually request a read for that
// particular index.
type LazyTransactionReader struct {
	lcm        xdr.LedgerCloseMeta
	start      int    // read-only
	passphrase string // read-only

	// we keep a mapping of hashes to envelope refs in the ledger meta, this is
	// fine since both have the same scope and we don't want to unnecessarily
	// keep two copies
	envelopesByHash map[xdr.Hash]*xdr.TransactionEnvelope
	transactions    []LedgerTransaction // cached for Rewind() calls
	lastRead        int                 // cycles through ^
}

// NewLazyTransactionReader creates a new reader instance from raw
// xdr.LedgerCloseMeta starting at a particular transaction index (0-based).
// Note that LazyTransactionReader is not thread safe and should not be shared
// by multiple goroutines.
func NewLazyTransactionReader(
	ledgerCloseMeta xdr.LedgerCloseMeta,
	passphrase string,
	start int,
) (*LazyTransactionReader, error) {
	if start >= ledgerCloseMeta.CountTransactions() || start < 0 {
		return nil, errors.New("'start' index exceeds ledger transaction count")
	}

	if ledgerCloseMeta.ProtocolVersion() < 20 {
		return nil, errors.New("LazyTransactionReader only works from Protocol 20 onward")
	}

	lazy := &LazyTransactionReader{
		lcm:        ledgerCloseMeta,
		passphrase: passphrase,
		start:      start,
		lastRead:   -1, // haven't started yet

		envelopesByHash: make(
			map[xdr.Hash]*xdr.TransactionEnvelope,
			ledgerCloseMeta.CountTransactions(),
		),
	}

	// See https://github.com/stellar/go/pull/2720: envelopes in the meta (which
	// just come straight from the agreed-upon transaction set) are not in the
	// same order as the actual list of metas (which are sorted by hash), so we
	// need to hash the envelopes *first* to properly associate them with their
	// metas.
	for _, txEnv := range ledgerCloseMeta.TransactionEnvelopes() {
		// we know that these are proper envelopes so errors aren't possible
		hash, _ := network.HashTransactionInEnvelope(txEnv, passphrase)
		lazy.envelopesByHash[xdr.Hash(hash)] = &txEnv
	}

	return lazy, nil
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

	// if it doesn't exist we're in BIG trouble elsewhere anyway
	envelope := reader.envelopesByHash[reader.lcm.TransactionHash(i)]
	reader.lastRead = i

	// Caching begins from `start`, so we need to properly offset into the
	// cached array to correlate the actual transaction index, which is by doing
	// `i-start`. We also have to have +txCount to fix negative offsets: mod (%)
	// in Go does not make it positive.
	cachedIdx := (i - reader.start + txCount) % txCount
	if cachedIdx < len(reader.transactions) { // cached? return immediately
		return reader.transactions[cachedIdx], nil
	}

	newTx := LedgerTransaction{
		Index:         uint32(i + 1), // Transactions start at '1'
		Envelope:      *envelope,
		Result:        reader.lcm.TransactionResultPair(i),
		UnsafeMeta:    reader.lcm.TxApplyProcessing(i),
		FeeChanges:    reader.lcm.FeeProcessing(i),
		LedgerVersion: uint32(reader.lcm.LedgerHeaderHistoryEntry().Header.LedgerVersion),
	}
	reader.transactions = append(reader.transactions, newTx)
	return newTx, nil
}

// Rewind resets the reader back to the first transaction in the ledger,
// or to `start` if you did not initialize the instance with 0.
func (reader *LazyTransactionReader) Rewind() {
	reader.lastRead = -1
}

// Close should be called when reading is finished to clean up memory.
func (reader *LazyTransactionReader) Close() {
	reader.Rewind()
	reader.transactions = nil
}
