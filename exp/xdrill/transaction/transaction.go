package transaction

import (
	"github.com/stellar/go/ingest"
)

type Transaction struct {
	// Use ingest.LedgerTransaction to be used with TransactionReader
	ingest.LedgerTransaction
}

// TODO: create low level helper functions
