package ingest

import (
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
			).Suffix("ON CONFLICT (id) DO UPDATE SET (amount, num_accounts, flags, toml) = (?, ?, ?, ?)",
				assetStat.Amount,
				assetStat.NumAccounts,
				assetStat.Flags,
				assetStat.Toml,
			)
		}
	}

	if hasValue {
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

	// TODO NNS 1
	return &AssetStat{
		assetID,
		10,
		5,
		0,
		"https://www.stellar.org/.well-known/stellar.toml",
	}
}
