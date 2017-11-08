package ingest

import (
	"strings"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/xdr"
)

// AssetStat is a row in the asset_stats table representing the stats per Asset
type AssetStat struct {
	ID          int64  `db:"id"`
	Amount      int64  `db:"amount"`
	NumAccounts int32  `db:"num_accounts"`
	Flags       int8   `db:"flags"`
	Toml        string `db:"toml"`
}

// UpdateAssetStats updates the db with the latest asset stats for the assetsModified
func UpdateAssetStats(is *Session, assetsModified *map[string]xdr.Asset) {
	hasValue := false
	for _, asset := range *assetsModified {
		assetStat := computeAssetStat(is, &asset)
		if is.Err != nil {
			return
		}

		if assetStat != nil {
			hasValue = true
			is.Ingestion.assetStats = is.Ingestion.assetStats.Values(
				assetStat.ID,
				assetStat.Amount,
				assetStat.NumAccounts,
				assetStat.Flags,
				assetStat.Toml,
			)
		}
	}

	if hasValue {
		is.Ingestion.assetStats = is.Ingestion.assetStats.
			Suffix("ON CONFLICT (id) DO UPDATE SET (amount, num_accounts, flags, toml) = (excluded.amount, excluded.num_accounts, excluded.flags, excluded.toml)")
		_, is.Err = is.Ingestion.DB.Exec(is.Ingestion.assetStats)
	}
}

func computeAssetStat(is *Session, asset *xdr.Asset) *AssetStat {
	if asset.Type == xdr.AssetTypeAssetTypeNative {
		return nil
	}

	assetID, err := is.Ingestion.GetOrInsertAssetID(*asset)
	if err != nil {
		is.Err = err
		return nil
	}

	var assetType xdr.AssetType
	var assetCode, assetIssuer string
	err = asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		is.Err = err
		return nil
	}

	coreQ := &core.Q{Session: is.Cursor.DB}

	numAccounts, amount, err := statTrustlinesInfo(coreQ, int32(assetType), assetCode, assetIssuer)
	if err != nil {
		is.Err = err
		return nil
	}

	flags, toml, err := statAccountInfo(coreQ, assetIssuer)
	if err != nil {
		is.Err = err
		return nil
	}

	return &AssetStat{
		assetID,
		amount,
		numAccounts,
		flags,
		toml,
	}
}

// statTrustlinesInfo fetches all the stats from the trustlines table
func statTrustlinesInfo(coreQ *core.Q, assetType int32, assetCode string, assetIssuer string) (int32, int64, error) {
	var trustlines []core.Trustline
	err := coreQ.TrustlinesByAsset(&trustlines, assetType, assetCode, assetIssuer)
	if err != nil {
		return -1, -1, err
	}

	numAccounts := int32(len(trustlines))

	var amount int64
	for _, t := range trustlines {
		amount += int64(t.Balance)
	}

	return numAccounts, amount, nil
}

// statAccountInfo fetches all the stats from the accounts table
func statAccountInfo(coreQ *core.Q, accountID string) (int8, string, error) {
	var account core.Account
	err := coreQ.AccountByAddress(&account, accountID)
	if err != nil {
		return -1, "", err
	}

	var toml string
	if !account.HomeDomain.Valid {
		toml = ""
	} else {
		trimmed := strings.Trim(account.HomeDomain.String, " ")
		if trimmed != "" {
			toml = "https://" + account.HomeDomain.String + "/.well-known/stellar.toml"
		} else {
			toml = ""
		}
	}

	return int8(account.Flags), toml, nil
}
