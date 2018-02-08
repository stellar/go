// Package resource contains the type definitions for all of horizons
// response resources.
package resource

import (
	"time"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resource/base"
	"github.com/stellar/go/services/horizon/internal/resource/effects"
	"github.com/stellar/go/services/horizon/internal/resource/operations"
	"github.com/stellar/go/services/horizon/internal/render/hal"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"golang.org/x/net/context"
)

// KeyTypeNames maps from strkey version bytes into json string values to use in
// horizon responses.
var KeyTypeNames = map[strkey.VersionByte]string{
	strkey.VersionByteAccountID: "ed25519_public_key",
	strkey.VersionByteSeed:      "ed25519_secret_seed",
	strkey.VersionByteHashX:     "sha256_hash",
	strkey.VersionByteHashTx:    "preauth_tx",
}

// Account is the summary of an account
type Account struct {
	Links struct {
		Self         hal.Link `json:"self"`
		Transactions hal.Link `json:"transactions"`
		Operations   hal.Link `json:"operations"`
		Payments     hal.Link `json:"payments"`
		Effects      hal.Link `json:"effects"`
		Offers       hal.Link `json:"offers"`
		Trades       hal.Link `json:"trades"`
		Data         hal.Link `json:"data"`
	} `json:"_links"`

	HistoryAccount
	Sequence             string            `json:"sequence"`
	SubentryCount        int32             `json:"subentry_count"`
	InflationDestination string            `json:"inflation_destination,omitempty"`
	HomeDomain           string            `json:"home_domain,omitempty"`
	Thresholds           AccountThresholds `json:"thresholds"`
	Flags                AccountFlags      `json:"flags"`
	Balances             []Balance         `json:"balances"`
	Signers              []Signer          `json:"signers"`
	Data                 map[string]string `json:"data"`
}

// AccountFlags represents the state of an account's flags
type AccountFlags struct {
	AuthRequired  bool `json:"auth_required"`
	AuthRevocable bool `json:"auth_revocable"`
}

// AccountThresholds represents an accounts "thresholds", the numerical values
// needed to satisfy the authorization of a given operation.
type AccountThresholds struct {
	LowThreshold  byte `json:"low_threshold"`
	MedThreshold  byte `json:"med_threshold"`
	HighThreshold byte `json:"high_threshold"`
}

// Asset represents a single asset
type Asset base.Asset

// AssetStat represents the statistics for a single Asset
type AssetStat struct {
	Links struct {
		Toml hal.Link `json:"toml"`
	} `json:"_links"`

	base.Asset
	PT          string       `json:"paging_token"`
	Amount      string       `json:"amount"`
	NumAccounts int32        `json:"num_accounts"`
	Flags       AccountFlags `json:"flags"`
}

// Balance represents an account's holdings for a single currency type
type Balance struct {
	Balance string `json:"balance"`
	Limit   string `json:"limit,omitempty"`
	base.Asset
}

// HistoryAccount is a simple resource, used for the account collection actions.
// It provides only the "TotalOrderID" of the account and its account id.
type HistoryAccount struct {
	ID        string `json:"id"`
	PT        string `json:"paging_token"`
	AccountID string `json:"account_id"`
}

// Ledger represents a single closed ledger
type Ledger struct {
	Links struct {
		Self         hal.Link `json:"self"`
		Transactions hal.Link `json:"transactions"`
		Operations   hal.Link `json:"operations"`
		Payments     hal.Link `json:"payments"`
		Effects      hal.Link `json:"effects"`
	} `json:"_links"`
	ID               string    `json:"id"`
	PT               string    `json:"paging_token"`
	Hash             string    `json:"hash"`
	PrevHash         string    `json:"prev_hash,omitempty"`
	Sequence         int32     `json:"sequence"`
	TransactionCount int32     `json:"transaction_count"`
	OperationCount   int32     `json:"operation_count"`
	ClosedAt         time.Time `json:"closed_at"`
	TotalCoins       string    `json:"total_coins"`
	FeePool          string    `json:"fee_pool"`
	BaseFee          int32     `json:"base_fee_in_stroops"`
	BaseReserve      int32     `json:"base_reserve_in_stroops"`
	MaxTxSetSize     int32     `json:"max_tx_set_size"`
	ProtocolVersion  int32     `json:"protocol_version"`
	HeaderXDR        string    `json:"header_xdr"`
}

// Offer is the display form of an offer to trade currency.
type Offer struct {
	Links struct {
		Self       hal.Link `json:"self"`
		OfferMaker hal.Link `json:"offer_maker"`
	} `json:"_links"`

	ID      int64  `json:"id"`
	PT      string `json:"paging_token"`
	Seller  string `json:"seller"`
	Selling Asset  `json:"selling"`
	Buying  Asset  `json:"buying"`
	Amount  string `json:"amount"`
	PriceR  Price  `json:"price_r"`
	Price   string `json:"price"`
}

// OrderBookSummary represents a snapshot summary of a given order book
type OrderBookSummary struct {
	Bids    []PriceLevel `json:"bids"`
	Asks    []PriceLevel `json:"asks"`
	Selling Asset        `json:"base"`
	Buying  Asset        `json:"counter"`
}

// Path represents a single payment path.
type Path struct {
	SourceAssetType        string  `json:"source_asset_type"`
	SourceAssetCode        string  `json:"source_asset_code,omitempty"`
	SourceAssetIssuer      string  `json:"source_asset_issuer,omitempty"`
	SourceAmount           string  `json:"source_amount"`
	DestinationAssetType   string  `json:"destination_asset_type"`
	DestinationAssetCode   string  `json:"destination_asset_code,omitempty"`
	DestinationAssetIssuer string  `json:"destination_asset_issuer,omitempty"`
	DestinationAmount      string  `json:"destination_amount"`
	Path                   []Asset `json:"path"`
}

// Price represents a price
type Price base.Price

// PriceLevel represents an aggregation of offers that share a given price
type PriceLevel struct {
	PriceR Price  `json:"price_r"`
	Price  string `json:"price"`
	Amount string `json:"amount"`
}

// Root is the initial map of links into the api.
type Root struct {
	Links struct {
		Account             hal.Link `json:"account"`
		AccountTransactions hal.Link `json:"account_transactions"`
		Assets              hal.Link `json:"assets"`
		Friendbot           hal.Link `json:"friendbot"`
		Metrics             hal.Link `json:"metrics"`
		OrderBook           hal.Link `json:"order_book"`
		Self                hal.Link `json:"self"`
		Transaction         hal.Link `json:"transaction"`
		Transactions        hal.Link `json:"transactions"`
	} `json:"_links"`

	HorizonVersion       string `json:"horizon_version"`
	StellarCoreVersion   string `json:"core_version"`
	HorizonSequence      int32  `json:"history_latest_ledger"`
	HistoryElderSequence int32  `json:"history_elder_ledger"`
	CoreSequence         int32  `json:"core_latest_ledger"`
	NetworkPassphrase    string `json:"network_passphrase"`
	ProtocolVersion      int32  `json:"protocol_version"`
}

// Signer represents one of an account's signers.
type Signer struct {
	PublicKey string `json:"public_key"`
	Weight    int32  `json:"weight"`
	Key       string `json:"key"`
	Type      string `json:"type"`
}

// Trade represents a horizon digested trade
type Trade struct {
	Links struct {
		Self      hal.Link `json:"self"`
		Base      hal.Link `json:"base"`
		Counter   hal.Link `json:"counter"`
		Operation hal.Link `json:"operation"`
	} `json:"_links"`

	ID                 string    `json:"id"`
	PT                 string    `json:"paging_token"`
	LedgerCloseTime    time.Time `json:"ledger_close_time"`
	OfferID            string    `json:"offer_id"`
	BaseAccount        string    `json:"base_account"`
	BaseAmount         string    `json:"base_amount"`
	BaseAssetType      string    `json:"base_asset_type"`
	BaseAssetCode      string    `json:"base_asset_code,omitempty"`
	BaseAssetIssuer    string    `json:"base_asset_issuer,omitempty"`
	CounterAccount     string    `json:"counter_account"`
	CounterAmount      string    `json:"counter_amount"`
	CounterAssetType   string    `json:"counter_asset_type"`
	CounterAssetCode   string    `json:"counter_asset_code,omitempty"`
	CounterAssetIssuer string    `json:"counter_asset_issuer,omitempty"`
	BaseIsSeller       bool      `json:"base_is_seller"`
	Price              xdr.Price `json:"price"`
}

// TradeEffect represents a trade effect resource.  NOTE (scott, 2017-12-08):
// this resource is being added back in temporarily to deal with a deploy snafu.
// I didn't properly message the community that we were changing the response
// format, and so we're adding this back in to allow transition.
type TradeEffect struct {
	Links struct {
		Self      hal.Link `json:"self"`
		Seller    hal.Link `json:"seller"`
		Buyer     hal.Link `json:"buyer"`
		Operation hal.Link `json:"operation"`
	} `json:"_links"`

	ID                string    `json:"id"`
	PT                string    `json:"paging_token"`
	OfferID           string    `json:"offer_id"`
	Seller            string    `json:"seller"`
	SoldAmount        string    `json:"sold_amount"`
	SoldAssetType     string    `json:"sold_asset_type"`
	SoldAssetCode     string    `json:"sold_asset_code,omitempty"`
	SoldAssetIssuer   string    `json:"sold_asset_issuer,omitempty"`
	Buyer             string    `json:"buyer"`
	BoughtAmount      string    `json:"bought_amount"`
	BoughtAssetType   string    `json:"bought_asset_type"`
	BoughtAssetCode   string    `json:"bought_asset_code,omitempty"`
	BoughtAssetIssuer string    `json:"bought_asset_issuer,omitempty"`
	LedgerCloseTime   time.Time `json:"created_at"`
}

// Transaction represents trade data aggregation over a period of time
type TradeAggregation struct {
	Timestamp     int64     `json:"timestamp"`
	TradeCount    int64     `json:"trade_count"`
	BaseVolume    string    `json:"base_volume"`
	CounterVolume string    `json:"counter_volume"`
	Average       string    `json:"avg"`
	High          string    `json:"high"`
	HighR         xdr.Price `json:"high_r"`
	Low           string    `json:"low"`
	LowR          xdr.Price `json:"low_r"`
	Open          string    `json:"open"`
	OpenR         xdr.Price `json:"open_r"`
	Close         string    `json:"close"`
	CloseR        xdr.Price `json:"close_r"`
}

// Transaction represents a single, successful transaction
type Transaction struct {
	Links struct {
		Self       hal.Link `json:"self"`
		Account    hal.Link `json:"account"`
		Ledger     hal.Link `json:"ledger"`
		Operations hal.Link `json:"operations"`
		Effects    hal.Link `json:"effects"`
		Precedes   hal.Link `json:"precedes"`
		Succeeds   hal.Link `json:"succeeds"`
	} `json:"_links"`
	ID              string    `json:"id"`
	PT              string    `json:"paging_token"`
	Hash            string    `json:"hash"`
	Ledger          int32     `json:"ledger"`
	LedgerCloseTime time.Time `json:"created_at"`
	Account         string    `json:"source_account"`
	AccountSequence string    `json:"source_account_sequence"`
	FeePaid         int32     `json:"fee_paid"`
	OperationCount  int32     `json:"operation_count"`
	EnvelopeXdr     string    `json:"envelope_xdr"`
	ResultXdr       string    `json:"result_xdr"`
	ResultMetaXdr   string    `json:"result_meta_xdr"`
	FeeMetaXdr      string    `json:"fee_meta_xdr"`
	MemoType        string    `json:"memo_type"`
	Memo            string    `json:"memo,omitempty"`
	Signatures      []string  `json:"signatures"`
	ValidAfter      string    `json:"valid_after,omitempty"`
	ValidBefore     string    `json:"valid_before,omitempty"`
}

// TransactionResultCodes represent a summary of result codes returned from
// a single xdr TransactionResult
type TransactionResultCodes struct {
	TransactionCode string   `json:"transaction"`
	OperationCodes  []string `json:"operations,omitempty"`
}

// TransactionSuccess represents the result of a successful transaction
// submission.
type TransactionSuccess struct {
	Links struct {
		Transaction hal.Link `json:"transaction"`
	} `json:"_links"`
	Hash   string `json:"hash"`
	Ledger int32  `json:"ledger"`
	Env    string `json:"envelope_xdr"`
	Result string `json:"result_xdr"`
	Meta   string `json:"result_meta_xdr"`
}

// NewEffect returns a resource of the appropriate sub-type for the provided
// effect record.
func NewEffect(
	ctx context.Context,
	row history.Effect,
) (result hal.Pageable, err error) {
	return effects.New(ctx, row)
}

// NewOperation returns a resource of the appropriate sub-type for the provided
// operation record.
func NewOperation(
	ctx context.Context,
	row history.Operation,
	ledger history.Ledger,
) (result hal.Pageable, err error) {
	return operations.New(ctx, row, ledger)
}

// KeyTypeFromAddress converts the version byte of the provided strkey encoded
// value (for example an account id or a signer key) and returns the appropriate
// horizon-specific type name.
func KeyTypeFromAddress(address string) (string, error) {
	vb, err := strkey.Version(address)
	if err != nil {
		return "", errors.Wrap(err, "invalid address")
	}

	result, ok := KeyTypeNames[vb]
	if !ok {
		result = "unknown"
	}

	return result, nil
}

// MustKeyTypeFromAddress is the panicking variant of KeyTypeFromAddress.
func MustKeyTypeFromAddress(address string) string {
	ret, err := KeyTypeFromAddress(address)
	if err != nil {
		panic(err)
	}

	return ret
}
