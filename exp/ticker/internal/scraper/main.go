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
	Code            string `toml:"code"`
	Issuer          string `toml:"issuer"`
	AnchorAsset     string `toml:"anchor_asset"`
	AnchorAssetType string `toml:"anchor_asset_type"`
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
}

// TOMLAsset is the interface to represent the aggregated Asset data.
type TOMLAsset struct {
	Code                    string     `json:"code"`
	Issuer                  string     `json:"issuer"`
	Type                    string     `json:"type"`
	NumAccounts             int32      `json:"num_accounts"`
	AuthRequired            bool       `json:"auth_required"`
	AuthRevocable           bool       `json:"auth_revocable"`
	Amount                  float64    `json:"amount"`
	IssuerDetails           TOMLIssuer `json:"-"`
	AssetControlledByDomain bool       `json:"asset_controlled_by_domain"`
	Error                   string     `json:"error"`
	AnchorAsset             string     `json:"anchor_asset"`
	AnchorAssetType         string     `json:"anchor_asset_type"`
	IsValid                 bool       `json:"is_valid"`
	LastValid               time.Time  `json:"last_valid"`
	LastChecked             time.Time  `json:"last_checked"`
	IsTrash                 bool       `json:"-"`
}

// FetchAllAssets fetches assets from the Horizon public net. If limit = 0, will fetch all assets.
func FetchAllAssets(c *horizonclient.Client, limit int, parallelism int) (assets []TOMLAsset, err error) {
	dirtyAssets, err := retrieveAssets(c, limit)
	if err != nil {
		return
	}

	assets, numTrash := parallelCleanUpAssets(dirtyAssets, parallelism)

	fmt.Printf(
		"Scanned %d entries. Trash: %d. Non-trash: %d\n",
		len(dirtyAssets),
		numTrash,
		len(assets),
	)
	return
}
