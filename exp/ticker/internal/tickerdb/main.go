package tickerdb

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/support/db"
)

// TickerSession provides helper methods for making queries against `DB`.
type TickerSession struct {
	db.Session
}

// Asset represents an entry on the assets table
type Asset struct {
	ID                          string    `db:"id"`
	Code                        string    `db:"code"`
	PublicKey                   string    `db:"public_key"`
	Type                        string    `db:"type"`
	NumAccounts                 int32     `db:"num_accounts"`
	AuthRequired                bool      `db:"auth_required"`
	AuthRevocable               bool      `db:"auth_revocable"`
	Amount                      float64   `db:"amount"`
	AssetControlledByDomain     bool      `db:"asset_controlled_by_domain"`
	AnchorAssetCode             string    `db:"anchor_asset_code"`
	AnchorAssetType             string    `db:"anchor_asset_type"`
	IsValid                     bool      `db:"is_valid"`
	ValidationError             string    `db:"validation_error"`
	LastValid                   time.Time `db:"last_valid"`
	LastChecked                 time.Time `db:"last_checked"`
	DisplayDecimals             int       `db:"display_decimals"`
	Name                        string    `db:"name"`
	Desc                        string    `db:"desc"`
	Conditions                  string    `db:"conditions"`
	IsAssetAnchored             bool      `db:"is_asset_anchored"`
	FixedNumber                 int       `db:"fixed_number"`
	MaxNumber                   int       `db:"max_number"`
	IsUnlimited                 bool      `db:"is_unlimited"`
	RedemptionInstructions      string    `db:"redemption_instructions"`
	CollateralAddresses         string    `db:"collateral_addresses"`
	CollateralAddressSignatures string    `db:"collateral_address_signatures"`
	Countries                   string    `db:"countries"`
	Status                      string    `db:"status"`
}

// CreateSession returns a new TickerSession that connects to the given db settings
func CreateSession(driverName, dataSourceName string) (session TickerSession, err error) {
	dbconn, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return
	}

	session.DB = dbconn
	return
}
