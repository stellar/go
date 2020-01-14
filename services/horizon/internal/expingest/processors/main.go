package processors

import (
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

type PipelineContextKey string

const (
	IngestUpdateDatabase = PipelineContextKey("IngestUpdateDatabase")
	IngestUpdateState    = PipelineContextKey("IngestUpdateState")
)

type DatabaseProcessorActionType string

const (
	Accounts          DatabaseProcessorActionType = "Accounts"
	AccountsForSigner DatabaseProcessorActionType = "AccountsForSigner"
	Data              DatabaseProcessorActionType = "Data"
	Offers            DatabaseProcessorActionType = "Offers"
	TrustLines        DatabaseProcessorActionType = "TrustLines"
	All               DatabaseProcessorActionType = "All"
)

// DatabaseProcessor is a processor (both state and ledger) that's responsible
// for persisting ledger data used in expingest in a database. It's possible
// to create multiple procesors of this type but they all should share the same
// *history.Q object to share a common transaction. `Action` defines what each
// processor is responsible for.
type DatabaseProcessor struct {
	AccountsQ     history.QAccounts
	DataQ         history.QData
	SignersQ      history.QSigners
	OffersQ       history.QOffers
	TrustLinesQ   history.QTrustLines
	AssetStatsQ   history.QAssetStats
	Action        DatabaseProcessorActionType
	IngestVersion int
	// AssetStatSet is used in TrustLines processor
	AssetStatSet AssetStatSet
	// batchUpsertTrustLines is a slice of trust lines to upsert in batch
	batchUpsertTrustLines []xdr.LedgerEntry
	// batchUpsertAccounts is a slice of accounts to upsert in batch
	batchUpsertAccounts []xdr.LedgerEntry
}

// OrderbookProcessor is a processor (both state and ledger) that's responsible
// for updating orderbook graph with new/updated/removed offers. Orderbook graph
// can be later used for path finding.
type OrderbookProcessor struct {
	OrderBookGraph *orderbook.OrderBookGraph
}

// ContextFilter writes read objects only if a given key is present in the
// pipline context.
type ContextFilter struct {
	Key PipelineContextKey
}
