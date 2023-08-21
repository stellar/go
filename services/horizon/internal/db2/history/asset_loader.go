package history

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sort"

	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/support/db"
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
	sealed bool
	set    map[AssetKey]interface{}
	ids    map[AssetKey]int64
}

// NewAssetLoader will construct a new AssetLoader instance.
func NewAssetLoader() *AssetLoader {
	return &AssetLoader{
		sealed: false,
		set:    map[AssetKey]interface{}{},
		ids:    map[AssetKey]int64{},
	}
}

// GetFuture registers the given asset into the loader and
// returns a FutureAssetID which will hold the history asset id for
// the asset after Exec() is called.
func (a *AssetLoader) GetFuture(asset AssetKey) FutureAssetID {
	if a.sealed {
		panic(errSealed)
	}
	a.set[asset] = nil
	return FutureAssetID{
		asset:  asset,
		loader: a,
	}
}

// GetNow returns the history asset id for the given asset.
// GetNow should only be called on values which were registered by
// GetFuture() calls. Also, Exec() must be called before any GetNow
// call can succeed.
func (a *AssetLoader) GetNow(asset AssetKey) int64 {
	if id, ok := a.ids[asset]; !ok {
		panic(fmt.Errorf("asset %v not present", asset))
	} else {
		return id
	}
}

func (a *AssetLoader) lookupKeys(ctx context.Context, q *Q, keys []AssetKey) error {
	var rows []Asset
	for i := 0; i < len(keys); i += loaderLookupBatchSize {
		end := i + loaderLookupBatchSize
		if end > len(keys) {
			end = len(keys)
		}
		subset := keys[i:end]
		keyStrings := make([]string, 0, len(subset))
		for _, key := range subset {
			keyStrings = append(keyStrings, key.Type+"/"+key.Code+"/"+key.Issuer)
		}
		err := q.Select(ctx, &rows, sq.Select("*").From("history_assets").Where(sq.Eq{
			"concat(asset_type, '/', asset_code, '/', asset_issuer)": keyStrings,
		}))
		if err != nil {
			return errors.Wrap(err, "could not select assets")
		}

		for _, row := range rows {
			a.ids[AssetKey{
				Type:   row.Type,
				Code:   row.Code,
				Issuer: row.Issuer,
			}] = row.ID
		}
	}
	return nil
}

// Exec will look up all the history asset ids for the assets registered in the loader.
// If there are no history asset ids for a given set of assets, Exec will insert rows
// into the history_assets table.
func (a *AssetLoader) Exec(ctx context.Context, session db.SessionInterface) error {
	a.sealed = true
	if len(a.set) == 0 {
		return nil
	}
	q := &Q{session}
	keys := make([]AssetKey, 0, len(a.set))
	for key := range a.set {
		keys = append(keys, key)
	}
	// sort entries before inserting rows to prevent deadlocks on acquiring a ShareLock
	// https://github.com/stellar/go/issues/2370
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

	if err := a.lookupKeys(ctx, q, keys); err != nil {
		return err
	}

	assetTypes := make([]string, 0, len(a.set)-len(a.ids))
	assetCodes := make([]string, 0, len(a.set)-len(a.ids))
	assetIssuers := make([]string, 0, len(a.set)-len(a.ids))
	insert := 0
	for _, key := range keys {
		if _, ok := a.ids[key]; ok {
			continue
		}
		assetTypes = append(assetTypes, key.Type)
		assetCodes = append(assetCodes, key.Code)
		assetIssuers = append(assetIssuers, key.Issuer)
		keys[insert] = key
		insert++
	}
	if insert == 0 {
		return nil
	}
	keys = keys[:insert]

	err := bulkInsert(
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
	if err != nil {
		return err
	}

	return a.lookupKeys(ctx, q, keys)
}
