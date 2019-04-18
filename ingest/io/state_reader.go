package ingestio

import "github.com/stellar/go/xdr"

// StateReader interface placehoilder
type StateReader interface {
	GetSequence() uint32
	Read() (bool, xdr.LedgerEntry, error)
}
