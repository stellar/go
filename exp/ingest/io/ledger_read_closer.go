package io

import (
	"io"
	"sync"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerReadCloser provides convenient, streaming access to the transactions within a ledger.
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

// LedgerTransaction represents the data for a single transaction within a ledger.
type LedgerTransaction struct {
	Index    uint32
	Envelope xdr.TransactionEnvelope
	Result   xdr.TransactionResultPair
	Meta     xdr.TransactionMeta
}

// DBLedgerReadCloser is a database-backed implementation of the io.LedgerReadCloser interface.
type DBLedgerReadCloser struct {
	sequence     uint32
	backend      ledgerbackend.LedgerBackend
	header       xdr.LedgerHeaderHistoryEntry
	transactions []LedgerTransaction
	readIdx      int
	initOnce     sync.Once
	readMutex    sync.Mutex
}

// Ensure DBLedgerReadCloser implements LedgerReadCloser
var _ LedgerReadCloser = (*DBLedgerReadCloser)(nil)

// MakeLedgerReadCloser is a factory method for LedgerReadCloser.
func MakeLedgerReadCloser(sequence uint32, backend ledgerbackend.LedgerBackend) *DBLedgerReadCloser {
	return &DBLedgerReadCloser{
		sequence: sequence,
		backend:  backend,
	}
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (dblrc *DBLedgerReadCloser) GetSequence() uint32 {
	dblrc.initOnce.Do(func() { dblrc.init() })
	return dblrc.sequence
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (dblrc *DBLedgerReadCloser) GetHeader() xdr.LedgerHeaderHistoryEntry {
	dblrc.initOnce.Do(func() { dblrc.init() })
	return dblrc.header
}

// Read returns the next transaction in the ledger, ordered by tx number, each time it is called. When there
// are no more transactions to return, an EOF error is returned.
func (dblrc *DBLedgerReadCloser) Read() (LedgerTransaction, error) {
	dblrc.initOnce.Do(func() { dblrc.init() })
	if dblrc.readIdx < len(dblrc.transactions) {

		dblrc.readMutex.Lock()
		dblrc.readIdx++
		dblrc.readMutex.Unlock()

		return dblrc.transactions[dblrc.readIdx-1], nil
	}
	return LedgerTransaction{}, io.EOF
}

// Close moves the read pointer so that subsequent calls to Read() will return EOF.
func (dblrc *DBLedgerReadCloser) Close() error {
	dblrc.initOnce.Do(func() { dblrc.init() })
	dblrc.readIdx = len(dblrc.transactions)
	return nil
}

// Init pulls data from the backend to set this object up for use.
func (dblrc *DBLedgerReadCloser) init() error {
	exists, ledgerCloseMeta, err := dblrc.backend.GetLedger(dblrc.sequence)

	if err != nil {
		return errors.Wrap(err, "error reading ledger from backend")
	}
	if !exists {
		return errors.Wrap(err, "ledger was not found")
	}

	dblrc.header = ledgerCloseMeta.LedgerHeader

	dblrc.storeTransactions(ledgerCloseMeta)

	return nil
}

// storeTransactions maps the close meta data into a slice of LedgerTransaction structs, to provide
// a per-transaction view of the data when Read() is called.
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
