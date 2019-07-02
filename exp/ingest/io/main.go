package io

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var ErrNotFound = errors.New("not found")

// StateReadCloser interface placeholder
type StateReadCloser interface {
	GetSequence() uint32
	// Read should return next ledger entry. If there are no more
	// entries it should return `io.EOF` error.
	Read() (xdr.LedgerEntryChange, error)
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some entries available so reader can stop
	// streaming them.
	Close() error
}

// StateWriteCloser interface placeholder
type StateWriteCloser interface {
	// Write is used to pass ledger entry change to the next processor. It can return
	// `ErrClosedPipe` when the pipe between processors has been closed meaning
	// that next processor does not need more data. In such situation the current
	// processor can terminate as sending more entries to a `StateWriteCloser`
	// does not make sense (will not be read).
	Write(xdr.LedgerEntryChange) error
	// Close should be called when there are no more entries
	// to write.
	Close() error
}

// LedgerReadCloser provides convenient, streaming access to the transactions within a ledger.
type LedgerReadCloser interface {
	GetSequence() uint32
	GetHeader() xdr.LedgerHeaderHistoryEntry
	// Read should return the next transaction. If there are no more
	// transactions it should return `io.EOF` error.
	Read() (LedgerTransaction, error)
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some transactions available so reader can stop
	// streaming them.
	Close() error
}

// LedgerWriteCloser provides convenient, streaming access to the transactions within a ledger.
type LedgerWriteCloser interface {
	// Write is used to pass a transaction to the next processor. It can return
	// `io.ErrClosedPipe` when the pipe between processors has been closed meaning
	// that next processor does not need more data. In such situation the current
	// processor can terminate as sending more transactions to a `LedgerWriteCloser`
	// does not make sense (will not be read).
	Write(LedgerTransaction) error
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some transactions available so the reader can stop
	// streaming them.
	Close() error
}

// LedgerTransaction represents the data for a single transaction within a ledger.
type LedgerTransaction struct {
	Index      uint32
	Envelope   xdr.TransactionEnvelope
	Result     xdr.TransactionResultPair
	Meta       xdr.TransactionMeta
	FeeChanges xdr.LedgerEntryChanges
}
