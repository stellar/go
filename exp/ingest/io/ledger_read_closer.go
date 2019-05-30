package io

import "github.com/stellar/go/xdr"

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
