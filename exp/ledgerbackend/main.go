package ledgerbackend

import (
	"github.com/stellar/go/xdr"
)

type LedgerBackend interface {
	GetLatestLedgerSequence() (uint32, error)
	// GetLedger ...
	// The first returned value is false when ledger does not exist in a backend.
	GetLedger(sequence uint32) (bool, xdr.LedgerCloseMeta, error)
}
