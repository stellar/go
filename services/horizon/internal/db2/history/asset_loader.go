package history

import (
	"context"
	"database/sql/driver"
	"sort"

	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/errors"
)

type AssetKey struct {
	Type   string
	Code   string
	Issuer string
}

// FutureAssetID represents a future history asset.
// A FutureAssetID is created by an AssetLoader and
// the asset id is available after calling Exec() on
// the AssetLoader.
type FutureAssetID struct {
	asset  AssetKey
	loader *AssetLoader
}

// Value implements the database/sql/driver Valuer interface.
func (a FutureAssetID) Value() (driver.Value, error) {
	return a.loader.GetNow(a.asset), nil
}

// AssetLoader will map assets to their history
// asset ids. If there is no existing mapping for a given sset,
// the AssetLoader will insert into the history_assets table to
// establish a mapping.
type AssetLoader struct {
	loader[AssetKey, FutureAssetID]
}

// NewAssetLoader will construct a new AssetLoader instance.
func NewAssetLoader() *AssetLoader {
	l := &AssetLoader{
		loader: loader[AssetKey, FutureAssetID]{
			sealed: false,
			set:    set.Set[AssetKey]{},
			ids:    map[AssetKey]int64{},
			sort: func(keys []AssetKey) {
				sort.Slice(keys, func(i, j int) bool {
					if keys[i].Type < keys[j].Type {
						return true
					}
					if keys[i].Code < keys[j].Code {
						return true
					}
					if keys[i].Issuer < keys[j].Issuer {
						return true
					}
					return false
				})
			},
			insert: func(ctx context.Context, q *Q, keys []AssetKey) error {
				assetTypes := make([]string, 0, len(keys))
				assetCodes := make([]string, 0, len(keys))
				assetIssuers := make([]string, 0, len(keys))
				for _, key := range keys {
					assetTypes = append(assetTypes, key.Type)
					assetCodes = append(assetCodes, key.Code)
					assetIssuers = append(assetIssuers, key.Issuer)
				}
				return bulkInsert(
					ctx,
					q,
					"history_assets",
					[]string{"asset_code", "asset_type", "asset_issuer"},
					[]bulkInsertField{
						{
							name:    "asset_code",
							dbType:  "character varying(12)",
							objects: assetCodes,
						},
						{
							name:    "asset_issuer",
							dbType:  "character varying(56)",
							objects: assetIssuers,
						},
						{
							name:    "asset_type",
							dbType:  "character varying(64)",
							objects: assetTypes,
						},
					},
				)
			},
		},
	}
	l.fetchAndUpdate = func(ctx context.Context, q *Q, keys []AssetKey) error {
		keyStrings := make([]string, 0, len(keys))
		for _, key := range keys {
			keyStrings = append(keyStrings, key.Type+"/"+key.Code+"/"+key.Issuer)
		}

		var rows []Asset
		err := q.Select(ctx, &rows, sq.Select("*").From("history_assets").Where(sq.Eq{
			"concat(asset_type, '/', asset_code, '/', asset_issuer)": keyStrings,
		}))
		if err != nil {
			return errors.Wrap(err, "could not select assets")
		}

		for _, row := range rows {
			l.ids[AssetKey{
				Type:   row.Type,
				Code:   row.Code,
				Issuer: row.Issuer,
			}] = row.ID
		}

		return nil
	}
	l.newFuture = func(key AssetKey) FutureAssetID {
		return FutureAssetID{
			asset:  key,
			loader: l,
		}
	}

	return l
}
