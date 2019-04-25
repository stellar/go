package scraper

import (
	"context"
	"time"

	horizonclient "github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
	hlog "github.com/stellar/go/support/log"
)

type ScraperConfig struct {
	Client *horizonclient.Client
	Logger *hlog.Entry
	Ctx    *context.Context
}

// TOMLDoc is the interface for storing TOML Issuer Documentation.
// See: https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0001.md#currency-documentation
type TOMLDoc struct {
	OrgName    string `toml:"ORG_NAME"`
	OrgURL     string `toml:"ORG_URL"`
	OrgTwitter string `toml:"ORG_TWITTER"`
}

// TOMLCurrency is the interface for storing TOML Currency Information.
// See: https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0001.md#currency-documentation
type TOMLCurrency struct {
	Code                        string   `toml:"code"`
	Issuer                      string   `toml:"issuer"`
	IsAssetAnchored             bool     `toml:"is_asset_anchored"`
	AnchorAsset                 string   `toml:"anchor_asset"`
	AnchorAssetType             string   `toml:"anchor_asset_type"`
	DisplayDecimals             int      `toml:"display_decimals"`
	Name                        string   `toml:"name"`
	Desc                        string   `toml:"desc"`
	Conditions                  string   `toml:"conditions"`
	FixedNumber                 int      `toml:"fixed_number"`
	MaxNumber                   int      `toml:"max_number"`
	IsUnlimited                 bool     `toml:"is_unlimited"`
	RedemptionInstructions      string   `toml:"redemption_instructions"`
	CollateralAddresses         []string `toml:"collateral_addresses"`
	CollateralAddressSignatures []string `toml:"collateral_address_signatures"`
	Status                      string   `toml:"status"`
}

// TOMLIssuer is the interface for storing TOML Issuer Information.
// See: https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0001.md#currency-documentation
type TOMLIssuer struct {
	FederationServer string         `toml:"FEDERATION_SERVER"`
	AuthServer       string         `toml:"AUTH_SERVER"`
	TransferServer   string         `toml:"TRANSFER_SERVER"`
	WebAuthEndpoint  string         `toml:"WEB_AUTH_ENDPOINT"`
	SigningKey       string         `toml:"SIGNING_KEY"`
	DepositServer    string         `toml:"DEPOSIT_SERVER"` // for legacy purposes
	Documentation    TOMLDoc        `toml:"DOCUMENTATION"`
	Currencies       []TOMLCurrency `toml:"CURRENCIES"`
	TOMLURL          string         `toml:"-"`
}

// FinalAsset is the interface to represent the aggregated Asset data.
type FinalAsset struct {
	Code                        string     `json:"code"`
	Issuer                      string     `json:"issuer"`
	Type                        string     `json:"type"`
	NumAccounts                 int32      `json:"num_accounts"`
	AuthRequired                bool       `json:"auth_required"`
	AuthRevocable               bool       `json:"auth_revocable"`
	Amount                      float64    `json:"amount"`
	IssuerDetails               TOMLIssuer `json:"-"`
	AssetControlledByDomain     bool       `json:"asset_controlled_by_domain"`
	Error                       string     `json:"-"`
	AnchorAsset                 string     `json:"anchor_asset"`
	AnchorAssetType             string     `json:"anchor_asset_type"`
	IsValid                     bool       `json:"-"`
	LastValid                   time.Time  `json:"-"`
	LastChecked                 time.Time  `json:"-"`
	IsTrash                     bool       `json:"-"`
	DisplayDecimals             int        `json:"display_decimals"`
	Name                        string     `json:"name"`
	Desc                        string     `json:"desc"`
	Conditions                  string     `json:"conditions"`
	IsAssetAnchored             bool       `json:"is_asset_anchored"`
	FixedNumber                 int        `json:"fixed_number"`
	MaxNumber                   int        `json:"max_number"`
	IsUnlimited                 bool       `json:"is_unlimited"`
	RedemptionInstructions      string     `json:"redemption_instructions"`
	CollateralAddresses         []string   `json:"collateral_addresses"`
	CollateralAddressSignatures []string   `json:"collateral_address_signatures"`
	Countries                   string     `json:"countries"`
	Status                      string     `json:"status"`
}

// FetchAllAssets fetches assets from the Horizon public net. If limit = 0, will fetch all assets.
func (c *ScraperConfig) FetchAllAssets(limit int, parallelism int) (assets []FinalAsset, err error) {
	dirtyAssets, err := c.retrieveAssets(limit)
	if err != nil {
		return
	}

	assets, numTrash := c.parallelProcessAssets(dirtyAssets, parallelism)

	c.Logger.Infof(
		"Scanned %d entries. Trash: %d. Non-trash: %d\n",
		len(dirtyAssets),
		numTrash,
		len(assets),
	)
	return
}

// FetchAllTrades fetches all trades for a given period, respecting the limit. If limit = 0,
// will fetch all trades for that given period.
func (c *ScraperConfig) FetchAllTrades(since time.Time, limit int) (trades []hProtocol.Trade, err error) {
	c.Logger.Info("Fetching trades from Horizon")

	trades, err = c.retrieveTrades(since, limit)

	c.Logger.Info("Last close time ingested:", trades[len(trades)-1].LedgerCloseTime)
	c.Logger.Infof("Fetched: %d trades\n", len(trades))
	return
}

// StreamNewTrades streams trades directly from horizon and calls the handler function
// whenever a new trade appears.
func (c *ScraperConfig) StreamNewTrades(cursor string, h horizonclient.TradeHandler) error {
	c.Logger.Info("Starting to stream trades with cursor at:", cursor)
	return c.streamTrades(h, cursor)
}
