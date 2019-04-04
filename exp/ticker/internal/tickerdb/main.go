package tickerdb

import (
	"time"

	"github.com/stellar/go/support/db"
)

// TickerSession provides helper methods for making queries against `DB`.
type TickerSession struct {
	db.Session
}

// Asset represents an entry on the a"ssets database
type Asset struct {
	ID                      string    `db:"id"`
	Code                    string    `db:"code"`
	Issuer                  string    `db:"issuer"`
	Type                    string    `db:"type"`
	NumAccounts             int32     `db:"num_accounts"`
	AuthRequired            bool      `db:"auth_required"`
	AuthRevocable           bool      `db:"auth_revocable"`
	Amount                  float64   `db:"amount"`
	AssetControlledByDomain bool      `db:"asset_controlled_by_domain"`
	AnchorAssetCode         string    `db:"anchor_asset_code"`
	AnchorAssetType         string    `db:"anchor_asset_type"`
	IsValid                 bool      `db:"is_valid"`
	ValidationError         string    `db:"validation_error"`
	LastValid               time.Time `db:"last_valid"`
	LastChecked             time.Time `db:"last_checked"`
}
