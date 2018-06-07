package horizon

import (
	"github.com/stellar/go/support/render/hal"
	. "github.com/stellar/go/protocols/horizon"
)

// TradeAggregationsPage returns a list of aggregated trade records, aggregated by resolution
type TradeAggregationsPage struct {
	Links hal.Links `json:"_links"`
	Embedded struct {
		Records []TradeAggregation `json:"records"`
	} `json:"_embedded"`
}

// TradesPage returns a list of trade records
type TradesPage struct {
	Links hal.Links `json:"_links"`
	Embedded struct {
		Records []Trade `json:"records"`
	} `json:"_embedded"`
}

type OffersPage struct {
	Links hal.Links `json:"_links"`
	Embedded struct {
		Records []Offer `json:"records"`
	} `json:"_embedded"`
}

type Payment struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	PagingToken string `json:"paging_token"`

	Links struct {
		Transaction struct {
			Href string `json:"href"`
		} `json:"transaction"`
	} `json:"_links"`

	TransactionHash string `json:"transaction_hash"`
	SourceAccount   string `json:"source_account"`
	CreatedAt       string `json:"created_at"`

	// create_account and account_merge field
	Account string `json:"account"`

	// create_account fields
	Funder          string `json:"funder"`
	StartingBalance string `json:"starting_balance"`

	// account_merge fields
	Into string `json:into"`

	// payment/path_payment fields
	From        string `json:"from"`
	To          string `json:"to"`
	AssetType   string `json:"asset_type"`
	AssetCode   string `json:"asset_code"`
	AssetIssuer string `json:"asset_issuer"`
	Amount      string `json:"amount"`

	// transaction fields
	Memo struct {
		Type  string `json:"memo_type"`
		Value string `json:"memo"`
	}
}
