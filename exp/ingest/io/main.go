package io

import (
	"io"

	"github.com/stellar/go/xdr"
)

var EOF = io.EOF

// StateReader interface placeholder
type StateReader interface {
	GetSequence() uint32
	// Read should return next ledger entry. If there are no more
	// entries it should return `io.EOF` error.
	Read() (xdr.LedgerEntry, error)
}

// StateWriteCloser interface placeholder
type StateWriteCloser interface {
	Write(xdr.LedgerEntry) error
	// Close should be called when there are no more entries
	// to write.
	Close() error
}
