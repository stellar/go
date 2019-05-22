package io

import "github.com/stellar/go/xdr"

// StateReader interface placeholder
type StateReader interface {
	GetSequence() uint32
	Read() (bool, xdr.LedgerEntry, error)
}
