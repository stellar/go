package history

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sort"
	"strings"

	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/ordered"
	"github.com/stellar/go/xdr"
)

type AssetKey struct {
	Type   string
	Code   string
	Issuer string
}

func (key AssetKey) String() string {
	if key.Type == xdr.AssetTypeToString[xdr.AssetTypeAssetTypeNative] {
		return key.Type
	}
	return key.Type + "/" + key.Code + "/" + key.Issuer
}

// AssetKeyFromXDR constructs an AssetKey from an xdr asset
func AssetKeyFromXDR(asset xdr.Asset) AssetKey {
	return AssetKey{
		Type:   xdr.AssetTypeToString[asset.Type],
		Code:   strings.TrimRight(asset.GetCode(), "\x00"),
		Issuer: asset.GetIssuer(),
	}
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
	return a.loader.GetNow(a.asset)
}

// AssetLoader will map assets to their history
// asset ids. If there is no existing mapping for a given sset,
// the AssetLoader will insert into the history_assets table to
// establish a mapping.
type AssetLoader struct {
	sealed bool
	set    set.Set[AssetKey]
	ids    map[AssetKey]int64
	stats  LoaderStats
}

// NewAssetLoader will construct a new AssetLoader instance.
func NewAssetLoader() *AssetLoader {
	return &AssetLoader{
		sealed: false,
		set:    set.Set[AssetKey]{},
		ids:    map[AssetKey]int64{},
		stats:  LoaderStats{},
	}
}

// GetFuture registers the given asset into the loader and
// returns a FutureAssetID which will hold the history asset id for
// the asset after Exec() is called.
func (a *AssetLoader) GetFuture(asset AssetKey) FutureAssetID {
	if a.sealed {
		panic(errSealed)
	}
	a.set.Add(asset)
	return FutureAssetID{
		asset:  asset,
		loader: a,
	}
}

// GetNow returns the history asset id for the given asset.
// GetNow should only be called on values which were registered by
// GetFuture() calls. Also, Exec() must be called before any GetNow
// call can succeed.
func (a *AssetLoader) GetNow(asset AssetKey) (int64, error) {
	if !a.sealed {
		return 0, fmt.Errorf(`invalid asset loader state,  
		Exec was not called yet to properly seal and resolve %v id`, asset)
	}
	if internalID, ok := a.ids[asset]; !ok {
		return 0, fmt.Errorf(`asset loader id %v was not found`, asset)
	} else {
		return internalID, nil
	}
}

func (a *AssetLoader) lookupKeys(ctx context.Context, q *Q, keys []AssetKey) error {
	var rows []Asset
	for i := 0; i < len(keys); i += loaderLookupBatchSize {
		end := ordered.Min(len(keys), i+loaderLookupBatchSize)
		subset := keys[i:end]
		args := make([]interface{}, 0, 3*len(subset))
		placeHolders := make([]string, 0, len(subset))
		for _, key := range subset {
			args = append(args, key.Code, key.Type, key.Issuer)
			placeHolders = append(placeHolders, "(?, ?, ?)")
		}
		rawSQL := fmt.Sprintf(
			"SELECT * FROM  history_assets WHERE (asset_code, asset_type, asset_issuer) in (%s)",
			strings.Join(placeHolders, ", "),
		)
		err := q.SelectRaw(ctx, &rows, rawSQL, args...)
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

	if err := a.lookupKeys(ctx, q, keys); err != nil {
		return err
	}
	a.stats.Total += len(keys)

	assetTypes := make([]string, 0, len(a.set)-len(a.ids))
	assetCodes := make([]string, 0, len(a.set)-len(a.ids))
	assetIssuers := make([]string, 0, len(a.set)-len(a.ids))
	// sort entries before inserting rows to prevent deadlocks on acquiring a ShareLock
	// https://github.com/stellar/go/issues/2370
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
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
	a.stats.Inserted += insert

	return a.lookupKeys(ctx, q, keys)
}

// Stats returns the number of assets registered in the loader and the number of assets
// inserted into the history_assets table.
func (a *AssetLoader) Stats() LoaderStats {
	return a.stats
}

func (a *AssetLoader) Name() string {
	return "AssetLoader"
}

// AssetLoaderStub is a stub wrapper around AssetLoader which allows
// you to manually configure the mapping of assets to history asset ids
type AssetLoaderStub struct {
	Loader *AssetLoader
}

// NewAssetLoaderStub returns a new AssetLoaderStub instance
func NewAssetLoaderStub() AssetLoaderStub {
	return AssetLoaderStub{Loader: NewAssetLoader()}
}

// Insert updates the wrapped AssetLoaderStub so that the given asset
// address is mapped to the provided history asset id
func (a AssetLoaderStub) Insert(asset AssetKey, id int64) {
	a.Loader.sealed = true
	a.Loader.ids[asset] = id
}
