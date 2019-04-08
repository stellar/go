package scraper

import (
	"fmt"
	"time"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
)

// TOMLDoc is the interface for storing TOML Issuer Documentation.
// See: https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0001.md#currency-documentation
type TOMLDoc struct {
	OrgURL string `toml:"ORG_URL"`
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
	OrgTwitter       string         `toml:"ORG_TWITTER"`
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
	Error                       string     `json:"error"`
	AnchorAsset                 string     `json:"anchor_asset"`
	AnchorAssetType             string     `json:"anchor_asset_type"`
	IsValid                     bool       `json:"is_valid"`
	LastValid                   time.Time  `json:"last_valid"`
	LastChecked                 time.Time  `json:"last_checked"`
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
func FetchAllAssets(c *horizonclient.Client, limit int, parallelism int) (assets []FinalAsset, err error) {
	dirtyAssets, err := retrieveAssets(c, limit)
	if err != nil {
		return
	}

	assets, numTrash := parallelProcessAssets(dirtyAssets, parallelism)

	fmt.Printf(
		"Scanned %d entries. Trash: %d. Non-trash: %d\n",
		len(dirtyAssets),
		numTrash,
		len(assets),
	)
	return
}
