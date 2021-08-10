package history

import (
	"context"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LiquidityPoolsQuery is a helper struct to configure queries to claimable balances
type LiquidityPoolsQuery struct {
	PageQuery db2.PageQuery
	Asset     *xdr.Asset
	Sponsor   *xdr.AccountId
	Claimant  *xdr.AccountId
}

// Cursor validates and returns the query page cursor
func (cbq LiquidityPoolsQuery) Cursor() (int64, *xdr.PoolId, error) {
	p := cbq.PageQuery
	var l int64
	var r *xdr.PoolId
	var err error

	if p.Cursor != "" {
		parts := strings.SplitN(p.Cursor, "-", 2)
		if len(parts) != 2 {
			return l, r, errors.New("Invalid cursor")
		}

		l, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return l, r, errors.Wrap(err, "Invalid cursor - first value should be higher than 0")
		}

		var balanceID xdr.PoolId
		if err = xdr.SafeUnmarshalHex(parts[1], &balanceID); err != nil {
			return l, r, errors.Wrap(err, "Invalid cursor - second value should be a valid claimable balance id")
		}
		r = &balanceID
		if l < 0 {
			return l, r, errors.Wrap(err, "Invalid cursor - first value should be higher than 0")
		}
	}

	return l, r, nil
}

// ApplyCursor applies cursor to the given sql. For performance reason the limit
// is not apply here. This allows us to hint the planner later to use the right
// indexes.
func (cbq LiquidityPoolsQuery) ApplyCursor(sql sq.SelectBuilder) (sq.SelectBuilder, error) {
	p := cbq.PageQuery
	l, r, err := cbq.Cursor()
	if err != nil {
		return sql, err
	}

	switch p.Order {
	case db2.OrderAscending:
		if l > 0 && r != nil {
			sql = sql.
				Where(sq.Expr("(cb.last_modified_ledger, cb.id) > (?, ?)", l, r))
		}
		sql = sql.OrderBy("cb.last_modified_ledger asc, cb.id asc")
	case db2.OrderDescending:
		if l > 0 && r != nil {
			sql = sql.
				Where(sq.Expr("(cb.last_modified_ledger, cb.id) < (?, ?)", l, r))
		}

		sql = sql.OrderBy("cb.last_modified_ledger desc, cb.id desc")
	default:
		return sql, errors.Errorf("invalid order: %s", p.Order)
	}

	return sql, nil
}

// LiquidityPool is a row of data from the `liquidity_pools` joined with table `liquidity_pool_assets`.
type LiquidityPool struct {
	PoolID             xdr.PoolId            `db:"id"`
	Type               xdr.LiquidityPoolType `db:"type"`
	Fee                uint32                `db:"fee"`
	TrustlineCount     uint64                `db:"trustline_count"`
	ShareCount         uint64                `db:"share_count"`
	Sponsor            null.String           `db:"sponsor"`
	LastModifiedLedger uint32                `db:"last_modified_ledger"`
	Assets             []LiquidityPoolAsset
}

type LiquidityPoolAsset struct {
	Asset   xdr.Asset `db:"asset"`
	Reserve uint64    `db:"reserve"`
}

type LiquidityPoolsBatchInsertBuilder interface {
	Add(ctx context.Context, entry *xdr.LedgerEntry) error
	Exec(ctx context.Context) error
}

// QLiquidityPools defines account related queries.
type QLiquidityPools interface {
	NewLiquidityPoolsBatchInsertBuilder(maxBatchSize int) LiquidityPoolsBatchInsertBuilder
	UpdateLiquidityPool(ctx context.Context, entry xdr.LedgerEntry) (int64, error)
	RemoveLiquidityPool(ctx context.Context, pool xdr.LiquidityPoolEntry) (int64, error)
	GetLiquidityPoolsByID(ctx context.Context, ids []xdr.PoolId) ([]LiquidityPool, error)
	CountLiquidityPools(ctx context.Context) (int, error)
}

// NewLiquidityPoolsBatchInsertBuilder constructs a new LiquidityPoolsBatchInsertBuilder instance
func (q *Q) NewLiquidityPoolsBatchInsertBuilder(maxBatchSize int) LiquidityPoolsBatchInsertBuilder {
	return &claimableBalancesBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("claimable_balances"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// CountLiquidityPools returns the total number of claimable balances in the DB
func (q *Q) CountLiquidityPools(ctx context.Context) (int, error) {
	sql := sq.Select("count(*)").From("claimable_balances")

	var count int
	if err := q.Get(ctx, &count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetLiquidityPoolsByID finds all claimable balances by PoolId
func (q *Q) GetLiquidityPoolsByID(ctx context.Context, ids []xdr.PoolId) ([]LiquidityPool, error) {
	var cBalances []LiquidityPool
	sql := selectLiquidityPools.Where(map[string]interface{}{"cb.id": ids})
	err := q.Select(ctx, &cBalances, sql)
	return cBalances, err
}

// UpdateLiquidityPool updates a row in the claimable_balances table.
// The only updatable value on claimable_balances is sponsor
// Returns number of rows affected and error.
func (q *Q) UpdateLiquidityPool(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	lPool := entry.Data.MustLiquidityPool()
	cBalanceMap := map[string]interface{}{
		"last_modified_ledger": entry.LastModifiedLedgerSeq,
		"sponsor":              ledgerEntrySponsorToNullString(entry),
	}

	sql := sq.Update("claimable_balances").SetMap(cBalanceMap).Where("id = ?", lPool.LiquidityPoolId)
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveLiquidityPool deletes a row in the claimable_balances table.
// Returns number of rows affected and error.
func (q *Q) RemoveLiquidityPool(ctx context.Context, lPool xdr.LiquidityPoolEntry) (int64, error) {
	sql := sq.Delete("claimable_balances").
		Where(sq.Eq{"id": lPool.LiquidityPoolId})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// FindLiquidityPoolByID returns a claimable balance.
func (q *Q) FindLiquidityPoolByID(ctx context.Context, balanceID xdr.PoolId) (LiquidityPool, error) {
	var claimableBalance LiquidityPool
	sql := selectLiquidityPools.Limit(1).Where("cb.id = ?", balanceID)
	err := q.Get(ctx, &claimableBalance, sql)
	return claimableBalance, err
}

// GetLiquidityPools finds all claimable balances where accountID is one of the claimants
func (q *Q) GetLiquidityPools(ctx context.Context, query LiquidityPoolsQuery) ([]LiquidityPool, error) {
	sql, err := query.ApplyCursor(selectLiquidityPools)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	if query.Asset != nil {
		sql = sql.Where("cb.asset = ?", query.Asset)
	}

	if query.Sponsor != nil {
		sql = sql.Where("cb.sponsor = ?", query.Sponsor.Address())
	}

	if query.Claimant != nil {
		sql = sql.
			Where(`cb.claimants @> '[{"destination": "` + query.Claimant.Address() + `"}]'`)
	}

	// we need to use WITH syntax to force the query planner to use the right
	// indexes, otherwise when the limit is small, it will use an index scan
	// which will be very slow once we have millions of records
	sql = sql.
		Prefix("WITH cb AS (").
		Suffix(
			") select "+claimableBalancesSelectStatement+" from cb LIMIT ?",
			query.PageQuery.Limit,
		)

	var results []LiquidityPool
	if err := q.Select(ctx, &results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

type liquidityPoolsBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func buildLiquidityPools(claimants []xdr.Claimant) Claimants {
	hClaimants := Claimants{}
	for _, c := range claimants {
		xc := c.MustV0()
		hClaimants = append(hClaimants, Claimant{
			Destination: xc.Destination.Address(),
			Predicate:   xc.Predicate,
		})
	}

	return hClaimants
}

func (i *liquidityPoolsBatchInsertBuilder) Add(ctx context.Context, entry *xdr.LedgerEntry) error {
	lPool := entry.Data.MustLiquidityPool()
	row := LiquidityPool{
		// TODO
		PoolID:             lPool.LiquidityPoolId,
		Sponsor:            ledgerEntrySponsorToNullString(*entry),
		LastModifiedLedger: uint32(entry.LastModifiedLedgerSeq),
	}
	return i.builder.RowStruct(ctx, row)
}

func (i *liquidityPoolsBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx)
}

// TODO
var liquidityPoolsSelectStatement = "lp.id, " +
	"lp.claimants, " +
	"cb.asset, " +
	"cb.amount, " +
	"cb.sponsor, " +
	"lp.last_modified_ledger, " +
	"cb.flags"

var selectLiquidityPools = sq.Select(liquidityPoolsSelectStatement).From("liquidity_pools lp")
