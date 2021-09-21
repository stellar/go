package history

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LiquidityPoolsQuery is a helper struct to configure queries to liquidity pools
type LiquidityPoolsQuery struct {
	PageQuery db2.PageQuery
	Assets    []xdr.Asset
}

// LiquidityPool is a row of data from the `liquidity_pools`.
type LiquidityPool struct {
	PoolID             string                     `db:"id"`
	Type               xdr.LiquidityPoolType      `db:"type"`
	Fee                uint32                     `db:"fee"`
	TrustlineCount     uint64                     `db:"trustline_count"`
	ShareCount         uint64                     `db:"share_count"`
	AssetReserves      LiquidityPoolAssetReserves `db:"asset_reserves"`
	LastModifiedLedger uint32                     `db:"last_modified_ledger"`
	Deleted            bool                       `db:"deleted"`
}

type LiquidityPoolAssetReserves []LiquidityPoolAssetReserve

func (c LiquidityPoolAssetReserves) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *LiquidityPoolAssetReserves) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &c)
}

type LiquidityPoolAssetReserve struct {
	Asset   xdr.Asset
	Reserve uint64
}

// liquidityPoolAssetReserveJSON  is an intermediate representation to allow encoding assets as base64 when stored in the DB
type liquidityPoolAssetReserveJSON struct {
	Asset   string `json:"asset"`
	Reserve uint64 `json:"reserve,string"` // use string-encoding to avoid problems with pgx https://github.com/jackc/pgx/issues/289
}

func (lpar LiquidityPoolAssetReserve) MarshalJSON() ([]byte, error) {
	asset, err := xdr.MarshalBase64(lpar.Asset)
	if err != nil {
		return nil, err
	}
	return json.Marshal(liquidityPoolAssetReserveJSON{asset, lpar.Reserve})
}

func (lpar *LiquidityPoolAssetReserve) UnmarshalJSON(data []byte) error {
	var lparJSON liquidityPoolAssetReserveJSON
	if err := json.Unmarshal(data, &lparJSON); err != nil {
		return err
	}
	var asset xdr.Asset
	if err := xdr.SafeUnmarshalBase64(lparJSON.Asset, &asset); err != nil {
		return err
	}
	lpar.Reserve = lparJSON.Reserve
	lpar.Asset = asset
	return nil
}

// QLiquidityPools defines liquidity-pool-related queries.
type QLiquidityPools interface {
	UpsertLiquidityPools(ctx context.Context, lps []LiquidityPool) error
	GetLiquidityPoolsByID(ctx context.Context, poolIDs []string) ([]LiquidityPool, error)
	GetAllLiquidityPools(ctx context.Context) ([]LiquidityPool, error)
	CountLiquidityPools(ctx context.Context) (int, error)
	FindLiquidityPoolByID(ctx context.Context, liquidityPoolID string) (LiquidityPool, error)
	GetUpdatedLiquidityPools(ctx context.Context, newerThanSequence uint32) ([]LiquidityPool, error)
	CompactLiquidityPools(ctx context.Context, cutOffSequence uint32) (int64, error)
}

// UpsertLiquidityPools upserts a batch of liquidity pools  in the liquidity_pools table.
// There's currently no limit of the number of liquidity pools this method can
// accept other than 2GB limit of the query string length what should be enough
// for each ledger with the current limits.
func (q *Q) UpsertLiquidityPools(ctx context.Context, lps []LiquidityPool) error {
	var poolID, typ, fee, shareCount, trustlineCount,
		assetReserves, lastModifiedLedger, deleted []interface{}

	for _, lp := range lps {
		poolID = append(poolID, lp.PoolID)
		typ = append(typ, lp.Type)
		fee = append(fee, lp.Fee)
		trustlineCount = append(trustlineCount, lp.TrustlineCount)
		shareCount = append(shareCount, lp.ShareCount)
		assetReserves = append(assetReserves, lp.AssetReserves)
		lastModifiedLedger = append(lastModifiedLedger, lp.LastModifiedLedger)
		deleted = append(deleted, lp.Deleted)
	}

	upsertFields := []upsertField{
		{"id", "text", poolID},
		{"type", "smallint", typ},
		{"fee", "integer", fee},
		{"trustline_count", "bigint", trustlineCount},
		{"share_count", "bigint", shareCount},
		{"asset_reserves", "jsonb", assetReserves},
		{"last_modified_ledger", "integer", lastModifiedLedger},
		{"deleted", "boolean", deleted},
	}

	return q.upsertRows(ctx, "liquidity_pools", "id", upsertFields)
}

// CountLiquidityPools returns the total number of liquidity pools  in the DB
func (q *Q) CountLiquidityPools(ctx context.Context) (int, error) {
	sql := sq.Select("count(*)").Where("deleted = ?", false).From("liquidity_pools")

	var count int
	if err := q.Get(ctx, &count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetLiquidityPoolsByID finds all liquidity pools by PoolId
func (q *Q) GetLiquidityPoolsByID(ctx context.Context, poolIDs []string) ([]LiquidityPool, error) {
	var liquidityPools []LiquidityPool
	sql := selectLiquidityPools.Where("deleted = ?", false).
		Where(map[string]interface{}{"lp.id": poolIDs})
	err := q.Select(ctx, &liquidityPools, sql)
	return liquidityPools, err
}

// FindLiquidityPoolByID returns a liquidity pool.
func (q *Q) FindLiquidityPoolByID(ctx context.Context, liquidityPoolID string) (LiquidityPool, error) {
	var lp LiquidityPool
	sql := selectLiquidityPools.Limit(1).Where("deleted = ?", false).Where("lp.id = ?", liquidityPoolID)
	err := q.Get(ctx, &lp, sql)
	return lp, err
}

// GetLiquidityPools finds all liquidity pools where accountID is one of the claimants
func (q *Q) GetLiquidityPools(ctx context.Context, query LiquidityPoolsQuery) ([]LiquidityPool, error) {
	sql, err := query.PageQuery.ApplyRawTo(selectLiquidityPools, "lp.id")
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}
	sql = sql.Where("deleted = ?", false)

	for _, asset := range query.Assets {
		assetB64, err := xdr.MarshalBase64(asset)
		if err != nil {
			return nil, err
		}
		sql = sql.
			Where(`lp.asset_reserves @> '[{"asset": "` + assetB64 + `"}]'`)
	}

	var results []LiquidityPool
	if err := q.Select(ctx, &results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

func (q *Q) GetAllLiquidityPools(ctx context.Context) ([]LiquidityPool, error) {
	var results []LiquidityPool
	if err := q.Select(ctx, &results, selectLiquidityPools.Where("deleted = ?", false)); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

// GetUpdatedLiquidityPools returns all liquidity pools created, updated, or deleted after the given ledger sequence.
func (q *Q) GetUpdatedLiquidityPools(ctx context.Context, newerThanSequence uint32) ([]LiquidityPool, error) {
	var pools []LiquidityPool
	err := q.Select(ctx, &pools, selectLiquidityPools.Where("lp.last_modified_ledger > ?", newerThanSequence))
	return pools, err
}

// CompactLiquidityPools removes rows from the liquidity pools table which are marked for deletion.
func (q *Q) CompactLiquidityPools(ctx context.Context, cutOffSequence uint32) (int64, error) {
	sql := sq.Delete("liquidity_pools").
		Where("deleted = ?", true).
		Where("last_modified_ledger <= ?", cutOffSequence)

	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, errors.Wrap(err, "cannot delete offer rows")
	}

	if err = q.UpdateLiquidityPoolCompactionSequence(ctx, cutOffSequence); err != nil {
		return 0, errors.Wrap(err, "cannot update liquidity pool compaction sequence")
	}

	return result.RowsAffected()
}

var liquidityPoolsSelectStatement = "lp.id, " +
	"lp.type, " +
	"lp.fee, " +
	"lp.trustline_count, " +
	"lp.share_count, " +
	"lp.asset_reserves, " +
	"lp.deleted, " +
	"lp.last_modified_ledger"

var selectLiquidityPools = sq.Select(liquidityPoolsSelectStatement).From("liquidity_pools lp")
