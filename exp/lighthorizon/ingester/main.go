package ingester

import (
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/metaarchive"
	"github.com/stellar/go/xdr"
)

//
// LightHorizon data model
//

// Ingester combines a source of unpacked ledger metadata and a way to create a
// ingestion reader interface on top of it.
type Ingester interface {
	metaarchive.MetaArchive

	NewLedgerTransactionReader(
		ledgerCloseMeta xdr.SerializedLedgerCloseMeta,
	) (LedgerTransactionReader, error)
}

// For now, this mirrors the `ingest` library exactly, but it's replicated so
// that we can diverge in the future if necessary.
type LedgerTransaction struct {
	*ingest.LedgerTransaction
}

type LedgerTransactionReader interface {
	Read() (LedgerTransaction, error)
}
