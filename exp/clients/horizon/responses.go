package horizonclient

import (
	"encoding/json"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/render/hal"
)

// Deprecated: use protocols/horizon instead
type Problem struct {
	Type     string                     `json:"type"`
	Title    string                     `json:"title"`
	Status   int                        `json:"status"`
	Detail   string                     `json:"detail,omitempty"`
	Instance string                     `json:"instance,omitempty"`
	Extras   map[string]json.RawMessage `json:"extras,omitempty"`
}

// Deprecated: use protocols/horizon instead
type Root = hProtocol.Root

// Deprecated: use protocols/horizon instead
type Account = hProtocol.Account

// Deprecated: use protocols/horizon instead
type AccountFlags = hProtocol.AccountFlags

// Deprecated: use protocols/horizon instead
type AccountThresholds = hProtocol.AccountThresholds

// Deprecated: use protocols/horizon instead
type Asset = hProtocol.Asset

// Deprecated: use protocols/horizon instead
type Balance = hProtocol.Balance

// Deprecated: use protocols/horizon instead
type HistoryAccount = hProtocol.HistoryAccount

// Deprecated: use protocols/horizon instead
type Ledger = hProtocol.Ledger

// Deprecated: use render/hal instead
type Link = hal.Link

// Deprecated: use protocols/horizon instead
type Offer = hProtocol.Offer

// EffectsPageResponse contains page of effects returned by Horizon.
// Currently used by LoadAccountMergeAmount only.
type EffectsPage struct {
	Embedded struct {
		Records []Effect
	} `json:"_embedded"`
}

// EffectResponse contains effect data returned by Horizon.
// Currently used by LoadAccountMergeAmount only.
// To Do: Have a more generic Effect struct that supports all effects
type Effect struct {
	Type   string `json:"type"`
	Amount string `json:"amount"`
}

// TradeAggregationsPage returns a list of aggregated trade records, aggregated by resolution
type TradeAggregationsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []TradeAggregation `json:"records"`
	} `json:"_embedded"`
}

// Deprecated: use protocols/horizon instead
type TradeAggregation = hProtocol.TradeAggregation

// TradesPage returns a list of trade records
type TradesPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Trade `json:"records"`
	} `json:"_embedded"`
}

// Deprecated: use protocols/horizon instead
type Trade = hProtocol.Trade

// Deprecated: use protocols/horizon instead
type OrderBookSummary = hProtocol.OrderBookSummary

// Deprecated: use protocols/horizon instead
type TransactionSuccess = hProtocol.TransactionSuccess

// Deprecated: use protocols/horizon instead
type TransactionResultCodes = hProtocol.TransactionResultCodes

// Deprecated: use protocols/horizon instead
type Signer = hProtocol.Signer

// OffersPage returns a list of offers
type OffersPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Offer `json:"records"`
	} `json:"_embedded"`
}

type Payment struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	PagingToken string `json:"paging_token"`

	Links struct {
		Effects struct {
			Href string `json:"href"`
		} `json:"effects"`
		Transaction struct {
			Href string `json:"href"`
		} `json:"transaction"`
	} `json:"_links"`

	SourceAccount string `json:"source_account"`
	CreatedAt     string `json:"created_at"`

	// create_account and account_merge field
	Account string `json:"account"`

	// create_account fields
	Funder          string `json:"funder"`
	StartingBalance string `json:"starting_balance"`

	// account_merge fields
	Into string `json:"into"`

	// payment/path_payment fields
	From        string `json:"from"`
	To          string `json:"to"`
	AssetType   string `json:"asset_type"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	Amount      string `json:"amount"`

	// transaction fields
	TransactionHash string `json:"transaction_hash"`
	Memo            struct {
		Type  string `json:"memo_type"`
		Value string `json:"memo"`
	}
}

// Deprecated: use protocols/horizon instead
type Price = hProtocol.Price

// Deprecated: use protocols/horizon instead
type PriceLevel = hProtocol.PriceLevel

// Deprecated: use protocols/horizon instead
type Transaction = hProtocol.Transaction

type AccountData struct {
	Value string `json:"value"`
}

// Deprecated: use protocols/horizon instead
type AssetStat = hProtocol.AssetStat

// AssetsPage contains page of assets returned by Horizon.
type AssetsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []AssetStat
	} `json:"_embedded"`
}

// LedgersPage contains page of ledger information returned by Horizon
type LedgersPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Ledger
	} `json:"_embedded"`
}

// SingleMetric represents a metric with a single value
type SingleMetric struct {
	Value int `json:"value"`
}

// LogMetric represents metrics that are logged by horizon for each log level
type LogMetric struct {
	Rate15m  float64 `json:"15m.rate"`
	Rate1m   float64 `json:"1m.rate"`
	Rate5m   float64 `json:"5m.rate"`
	Count    int     `json:"count"`
	MeanRate float64 `json:"mean.rate"`
}

// LogTotalMetric represents total metrics logged for ingester, requests and submitted transactions
type LogTotalMetric struct {
	LogMetric
	Percent75   float64 `json:"75%"`
	Percent95   float64 `json:"95%"`
	Percent99   float64 `json:"99%"`
	Percent99_9 float64 `json:"99.9%"`
	Max         float64 `json:"max"`
	Mean        float64 `json:"mean"`
	Median      float64 `json:"median"`
	Min         float64 `json:"min"`
	StdDev      float64 `json:"stddev"`
}

// Metrics represents a response of metrics from horizon
type Metrics struct {
	Links                  hal.Links      `json:"_links"`
	GoRoutines             SingleMetric   `json:"goroutines"`
	HistoryElderLedger     SingleMetric   `json:"history.elder_ledger"`
	HistoryLatestLedger    SingleMetric   `json:"history.latest_ledger"`
	HistoryOpenConnections SingleMetric   `json:"history.open_connections"`
	IngesterIngestLedger   LogTotalMetric `json:"ingester.ingest_ledger"`
	IngesterClearLedger    LogTotalMetric `json:"ingester.clear_ledger"`
	LoggingDebug           LogMetric      `json:"logging.debug"`
	LoggingError           LogMetric      `json:"logging.error"`
	LoggingInfo            LogMetric      `json:"logging.info"`
	LoggingPanic           LogMetric      `json:"logging.panic"`
	LoggingWarning         LogMetric      `json:"logging.warning"`
	RequestsFailed         LogMetric      `json:"requests.failed"`
	RequestsSucceeded      LogMetric      `json:"requests.succeeded"`
	RequestsTotal          LogTotalMetric `json:"requests.total"`
	CoreLatestLedger       SingleMetric   `json:"stellar_core.latest_ledger"`
	CoreOpenConnections    SingleMetric   `json:"stellar_core.open_connections"`
	TxsubBuffered          SingleMetric   `json:"txsub.buffered"`
	TxsubFailed            LogMetric      `json:"txsub.failed"`
	TxsubOpen              SingleMetric   `json:"txsub.open"`
	TxsubSucceeded         LogMetric      `json:"txsub.succeeded"`
	TxsubTotal             LogTotalMetric `json:"txsub.total"`
}
