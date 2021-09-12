package history

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/db"
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

type LiquidityPoolsBatchInsertBuilder interface {
	Add(ctx context.Context, lp LiquidityPool) error
	Exec(ctx context.Context) error
}

// QLiquidityPools defines liquidity-pool-related queries.
type QLiquidityPools interface {
	NewLiquidityPoolsBatchInsertBuilder(maxBatchSize int) LiquidityPoolsBatchInsertBuilder
	UpdateLiquidityPool(ctx context.Context, lp LiquidityPool) (int64, error)
	RemoveLiquidityPool(ctx context.Context, liquidityPoolID string, lastModifiedLedger uint32) (int64, error)
	GetLiquidityPoolsByID(ctx context.Context, poolIDs []string) ([]LiquidityPool, error)
	GetAllLiquidityPools(ctx context.Context) ([]LiquidityPool, error)
	CountLiquidityPools(ctx context.Context) (int, error)
	FindLiquidityPoolByID(ctx context.Context, liquidityPoolID string) (LiquidityPool, error)
	GetUpdatedLiquidityPools(ctx context.Context, newerThanSequence uint32) ([]LiquidityPool, error)
	CompactLiquidityPools(ctx context.Context, cutOffSequence uint32) (int64, error)
}

// NewLiquidityPoolsBatchInsertBuilder constructs a new LiquidityPoolsBatchInsertBuilder instance
func (q *Q) NewLiquidityPoolsBatchInsertBuilder(maxBatchSize int) LiquidityPoolsBatchInsertBuilder {
	cols := db.ColumnsForStruct(LiquidityPool{})
	excludedCols := make([]string, len(cols))
	for i, col := range cols {
		excludedCols[i] = "EXCLUDED." + col
	}
	suffix := fmt.Sprintf(
		"ON CONFLICT (id) DO UPDATE SET (%s) = (%s)",
		strings.Join(cols, ", "),
		strings.Join(excludedCols, ", "),
	)
	return &liquidityPoolsBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("liquidity_pools"),
			MaxBatchSize: maxBatchSize,
			Suffix:       suffix,
		},
	}
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

// UpdateLiquidityPool updates a row in the liquidity_pools table.
// Returns number of rows affected and error.
func (q *Q) UpdateLiquidityPool(ctx context.Context, lp LiquidityPool) (int64, error) {
	updateBuilder := q.GetTable("liquidity_pools").Update()
	result, err := updateBuilder.SetStruct(lp, []string{}).Where("id = ?", lp.PoolID).Exec(ctx)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveLiquidityPool marks the given liquidity pool as deleted.
// Returns number of rows affected and error.
func (q *Q) RemoveLiquidityPool(ctx context.Context, liquidityPoolID string, lastModifiedLedger uint32) (int64, error) {
	sql := sq.Update("liquidity_pools").
		Set("deleted", true).
		Set("last_modified_ledger", lastModifiedLedger).
		Where(sq.Eq{"id": liquidityPoolID})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
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

type liquidityPoolsBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func (i *liquidityPoolsBatchInsertBuilder) Add(ctx context.Context, lp LiquidityPool) error {
	return i.builder.RowStruct(ctx, lp)
}

func (i *liquidityPoolsBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx)
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
