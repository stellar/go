package assets

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
)

// AssetStatsR is the result from the AssetStatsQ query
type AssetStatsR struct {
	ID          int64  `db:"id"`
	Type        string `db:"asset_type"`
	Code        string `db:"asset_code"`
	Issuer      string `db:"asset_issuer"`
	Amount      int64  `db:"amount"`
	NumAccounts int32  `db:"num_accounts"`
	Flags       int8   `db:"flags"`
	Toml        string `db:"toml"`
}

// AssetStatsQ is the query to fetch all assets in the system
type AssetStatsQ struct {
	AssetCode   *string
	AssetIssuer *string
	PageQuery   *db2.PageQuery
}

// GetSQL allows this query to be executed by the caller
func (q AssetStatsQ) GetSQL() (sq.SelectBuilder, error) {
	sql := selectQuery
	if q.AssetCode != nil && *q.AssetCode != "" {
		sql = sql.Where("hist.asset_code = ?", *q.AssetCode)
	}
	if q.AssetIssuer != nil && *q.AssetIssuer != "" {
		sql = sql.Where("hist.asset_issuer = ?", *q.AssetIssuer)
	}

	var err error
	if q.PageQuery != nil {
		sql, err = q.PageQuery.ApplyTo(sql, "hist.id")
		if err != nil {
			return sql, err
		}
	}
	sql = sql.OrderBy("hist.asset_code ASC", "hist.asset_issuer ASC")
	return sql, nil
}

var selectQuery = sq.
	Select(
		"hist.id as id",
		"hist.asset_type",
		"hist.asset_code",
		"hist.asset_issuer",
		"stats.amount",
		"stats.num_accounts",
		"stats.flags",
		"stats.toml",
	).
	From("history_assets hist").
	Join("asset_stats stats ON hist.id = stats.id")
