package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func assetStatToMap(assetStat ExpAssetStat) map[string]interface{} {
	return map[string]interface{}{
		"asset_type":   assetStat.AssetType,
		"asset_code":   assetStat.AssetCode,
		"asset_issuer": assetStat.AssetIssuer,
		"amount":       assetStat.Amount,
		"num_accounts": assetStat.NumAccounts,
	}
}

func assetStatToPrimaryKeyMap(assetStat ExpAssetStat) map[string]interface{} {
	return map[string]interface{}{
		"asset_type":   assetStat.AssetType,
		"asset_code":   assetStat.AssetCode,
		"asset_issuer": assetStat.AssetIssuer,
	}
}

// InsertAssetStats a set of asset stats into the exp_asset_stats
func (q *Q) InsertAssetStats(assetStats []ExpAssetStat, batchSize int) error {
	builder := &db.BatchInsertBuilder{
		Table:        q.GetTable("exp_asset_stats"),
		MaxBatchSize: batchSize,
	}

	for _, assetStat := range assetStats {
		if err := builder.Row(assetStatToMap(assetStat)); err != nil {
			return errors.Wrap(err, "could not insert asset assetStat row")
		}
	}

	if err := builder.Exec(); err != nil {
		return errors.Wrap(err, "could not exec asset assetStats insert builder")
	}

	return nil
}

// InsertAssetStat a single asset assetStat row into the exp_asset_stats
// Returns number of rows affected and error.
func (q *Q) InsertAssetStat(assetStat ExpAssetStat) (int64, error) {
	sql := sq.Insert("exp_asset_stats").SetMap(assetStatToMap(assetStat))
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// UpdateAssetStat updates a row in the exp_asset_stats table.
// Returns number of rows affected and error.
func (q *Q) UpdateAssetStat(assetStat ExpAssetStat) (int64, error) {
	sql := sq.Update("exp_asset_stats").
		SetMap(assetStatToMap(assetStat)).
		Where(assetStatToPrimaryKeyMap(assetStat))
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveAssetStat removes a row in the exp_asset_stats table.
func (q *Q) RemoveAssetStat(assetType xdr.AssetType, assetCode, assetIssuer string) (int64, error) {
	sql := sq.Delete("exp_asset_stats").
		Where(map[string]interface{}{
			"asset_type":   assetType,
			"asset_code":   assetCode,
			"asset_issuer": assetIssuer,
		})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetAssetStat returns a row in the exp_asset_stats table.
func (q *Q) GetAssetStat(assetType xdr.AssetType, assetCode, assetIssuer string) (ExpAssetStat, error) {
	sql := selectAssetStats.Where(map[string]interface{}{
		"asset_type":   assetType,
		"asset_code":   assetCode,
		"asset_issuer": assetIssuer,
	})
	var assetStat ExpAssetStat
	err := q.Get(&assetStat, sql)
	return assetStat, err
}

var selectAssetStats = sq.Select("exp_asset_stats.*").From("exp_asset_stats")
