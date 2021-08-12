package history

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LiquidityPoolsQuery is a helper struct to configure queries to liquidity pools
type LiquidityPoolsQuery struct {
	PageQuery db2.PageQuery
	Assets    []xdr.Asset
	Sponsor   *xdr.AccountId
}

// Cursor validates and returns the query page cursor
func (cbq LiquidityPoolsQuery) Cursor() (string, error) {
	p := cbq.PageQuery

	if p.Cursor != "" {
		// validate the cursor
		var poolID xdr.PoolId
		if err := xdr.SafeUnmarshalHex(p.Cursor, &poolID); err != nil {
			return "", errors.Wrap(err, "Invalid cursor - value should be a valid liquidity pool id")
		}
	}

	return p.Cursor, nil
}

// ApplyCursor applies cursor to the given sql. For performance reason the limit
// is not applied here. This allows us to hint the planner later to use the right
// indexes.
func (cbq LiquidityPoolsQuery) ApplyCursor(sql sq.SelectBuilder) (sq.SelectBuilder, error) {
	p := cbq.PageQuery
	r, err := cbq.Cursor()
	if err != nil {
		return sql, err
	}

	switch p.Order {
	case db2.OrderAscending:
		if r != "" {
			sql = sql.
				Where(sq.Expr("lp.id > ?", r))
		}
		sql = sql.OrderBy("lp.id asc")
	case db2.OrderDescending:
		if r != "" {
			sql = sql.
				Where(sq.Expr("lp.id < ?", r))
		}

		sql = sql.OrderBy("lp.id desc")
	default:
		return sql, errors.Errorf("invalid order: %s", p.Order)
	}

	return sql, nil
}

// LiquidityPool is a row of data from the `liquidity_pools`.
type LiquidityPool struct {
	PoolID             string                     `db:"id"`
	Type               xdr.LiquidityPoolType      `db:"type"`
	Fee                uint32                     `db:"fee"`
	TrustlineCount     uint64                     `db:"trustline_count"`
	ShareCount         uint64                     `db:"share_count"`
	AssetReserves      LiquidityPoolAssetReserves `db:"asset_reserves"`
	Sponsor            null.String                `db:"sponsor"`
	LastModifiedLedger uint32                     `db:"last_modified_ledger"`
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
	RemoveLiquidityPool(ctx context.Context, liquidityPoolID string) (int64, error)
	GetLiquidityPoolsByID(ctx context.Context, poolIDs []string) ([]LiquidityPool, error)
	CountLiquidityPools(ctx context.Context) (int, error)
}

// NewLiquidityPoolsBatchInsertBuilder constructs a new LiquidityPoolsBatchInsertBuilder instance
func (q *Q) NewLiquidityPoolsBatchInsertBuilder(maxBatchSize int) LiquidityPoolsBatchInsertBuilder {
	return &liquidityPoolsBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("liquidity_pools"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// CountLiquidityPools returns the total number of liquidity pools  in the DB
func (q *Q) CountLiquidityPools(ctx context.Context) (int, error) {
	sql := sq.Select("count(*)").From("liquidity_pools")

	var count int
	if err := q.Get(ctx, &count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetLiquidityPoolsByID finds all liquidity pools by PoolId
func (q *Q) GetLiquidityPoolsByID(ctx context.Context, poolIDs []string) ([]LiquidityPool, error) {
	var cBalances []LiquidityPool
	sql := selectLiquidityPools.Where(map[string]interface{}{"lp.id": poolIDs})
	err := q.Select(ctx, &cBalances, sql)
	return cBalances, err
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

// RemoveLiquidityPool deletes a row in the liquidity_pools table.
// Returns number of rows affected and error.
func (q *Q) RemoveLiquidityPool(ctx context.Context, liquidityPoolID string) (int64, error) {
	sql := sq.Delete("liquidity_pools").
		Where(sq.Eq{"id": liquidityPoolID})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// FindLiquidityPoolByID returns a liquidity pool.
func (q *Q) FindLiquidityPoolByID(ctx context.Context, liquidityPoolID string) (LiquidityPool, error) {
	var claimableBalance LiquidityPool
	sql := selectLiquidityPools.Limit(1).Where("lp.id = ?", liquidityPoolID)
	err := q.Get(ctx, &claimableBalance, sql)
	return claimableBalance, err
}

// GetLiquidityPools finds all liquidity pools where accountID is one of the claimants
func (q *Q) GetLiquidityPools(ctx context.Context, query LiquidityPoolsQuery) ([]LiquidityPool, error) {
	sql, err := query.ApplyCursor(selectLiquidityPools)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	for _, asset := range query.Assets {
		assetB64, err := xdr.MarshalBase64(asset)
		if err != nil {
			return nil, err
		}
		sql = sql.
			Where(`lp.asset_reserves @> '[{"asset": "` + assetB64 + `"}]'`)
	}

	if query.Sponsor != nil {
		sql = sql.Where("lp.sponsor = ?", query.Sponsor.Address())
	}

	var results []LiquidityPool
	if err := q.Select(ctx, &results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
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
	"lp.sponsor, " +
	"lp.last_modified_ledger"

var selectLiquidityPools = sq.Select(liquidityPoolsSelectStatement).From("liquidity_pools lp")
