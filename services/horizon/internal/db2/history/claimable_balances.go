package history

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ClaimableBalancesQuery is a helper struct to configure queries to claimable balances
type ClaimableBalancesQuery struct {
	PageQuery db2.PageQuery
	Asset     *xdr.Asset
	Sponsor   *xdr.AccountId
	Claimant  *xdr.AccountId
}

// Cursor validates and returns the query page cursor
func (cbq ClaimableBalancesQuery) Cursor() (int64, *xdr.ClaimableBalanceId, error) {
	p := cbq.PageQuery
	var l int64
	var r *xdr.ClaimableBalanceId
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

		var balanceID xdr.ClaimableBalanceId
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
func (cbq ClaimableBalancesQuery) ApplyCursor(sql sq.SelectBuilder) (sq.SelectBuilder, error) {
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

// ClaimableBalance is a row of data from the `claimable_balances` table.
type ClaimableBalance struct {
	BalanceID          xdr.ClaimableBalanceId `db:"id"`
	Claimants          Claimants              `db:"claimants"`
	Asset              xdr.Asset              `db:"asset"`
	Amount             xdr.Int64              `db:"amount"`
	Sponsor            null.String            `db:"sponsor"`
	LastModifiedLedger uint32                 `db:"last_modified_ledger"`
	Flags              uint32                 `db:"flags"`
}

type Claimants []Claimant

func (c Claimants) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *Claimants) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &c)
}

type Claimant struct {
	Destination string             `json:"destination"`
	Predicate   xdr.ClaimPredicate `json:"predicate"`
}

type ClaimableBalancesBatchInsertBuilder interface {
	Add(ctx context.Context, entry *xdr.LedgerEntry) error
	Exec(ctx context.Context) error
}

// QClaimableBalances defines account related queries.
type QClaimableBalances interface {
	NewClaimableBalancesBatchInsertBuilder(maxBatchSize int) ClaimableBalancesBatchInsertBuilder
	UpdateClaimableBalance(ctx context.Context, entry xdr.LedgerEntry) (int64, error)
	RemoveClaimableBalance(ctx context.Context, cBalance xdr.ClaimableBalanceEntry) (int64, error)
	GetClaimableBalancesByID(ctx context.Context, ids []xdr.ClaimableBalanceId) ([]ClaimableBalance, error)
	CountClaimableBalances(ctx context.Context) (int, error)
}

// NewClaimableBalancesBatchInsertBuilder constructs a new ClaimableBalancesBatchInsertBuilder instance
func (q *Q) NewClaimableBalancesBatchInsertBuilder(maxBatchSize int) ClaimableBalancesBatchInsertBuilder {
	return &claimableBalancesBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("claimable_balances"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// CountClaimableBalances returns the total number of claimable balances in the DB
func (q *Q) CountClaimableBalances(ctx context.Context) (int, error) {
	sql := sq.Select("count(*)").From("claimable_balances")

	var count int
	if err := q.Get(ctx, &count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetClaimableBalancesByID finds all claimable balances by ClaimableBalanceId
func (q *Q) GetClaimableBalancesByID(ctx context.Context, ids []xdr.ClaimableBalanceId) ([]ClaimableBalance, error) {
	var cBalances []ClaimableBalance
	sql := selectClaimableBalances.Where(map[string]interface{}{"cb.id": ids})
	err := q.Select(ctx, &cBalances, sql)
	return cBalances, err
}

// UpdateClaimableBalance updates a row in the claimable_balances table.
// The only updatable value on claimable_balances is sponsor
// Returns number of rows affected and error.
func (q *Q) UpdateClaimableBalance(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	cBalance := entry.Data.MustClaimableBalance()
	cBalanceMap := map[string]interface{}{
		"last_modified_ledger": entry.LastModifiedLedgerSeq,
		"sponsor":              ledgerEntrySponsorToNullString(entry),
	}

	sql := sq.Update("claimable_balances").SetMap(cBalanceMap).Where("id = ?", cBalance.BalanceId)
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveClaimableBalance deletes a row in the claimable_balances table.
// Returns number of rows affected and error.
func (q *Q) RemoveClaimableBalance(ctx context.Context, cBalance xdr.ClaimableBalanceEntry) (int64, error) {
	sql := sq.Delete("claimable_balances").
		Where(sq.Eq{"id": cBalance.BalanceId})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// FindClaimableBalanceByID returns a claimable balance.
func (q *Q) FindClaimableBalanceByID(ctx context.Context, balanceID xdr.ClaimableBalanceId) (ClaimableBalance, error) {
	var claimableBalance ClaimableBalance
	sql := selectClaimableBalances.Limit(1).Where("cb.id = ?", balanceID)
	err := q.Get(ctx, &claimableBalance, sql)
	return claimableBalance, err
}

// GetClaimableBalances finds all claimable balances where accountID is one of the claimants
func (q *Q) GetClaimableBalances(ctx context.Context, query ClaimableBalancesQuery) ([]ClaimableBalance, error) {
	sql, err := query.ApplyCursor(selectClaimableBalances)
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

	var results []ClaimableBalance
	if err := q.Select(ctx, &results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

type claimableBalancesBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func buildClaimants(claimants []xdr.Claimant) Claimants {
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

func (i *claimableBalancesBatchInsertBuilder) Add(ctx context.Context, entry *xdr.LedgerEntry) error {
	cBalance := entry.Data.MustClaimableBalance()
	row := ClaimableBalance{
		BalanceID:          cBalance.BalanceId,
		Claimants:          buildClaimants(cBalance.Claimants),
		Asset:              cBalance.Asset,
		Amount:             cBalance.Amount,
		Sponsor:            ledgerEntrySponsorToNullString(*entry),
		LastModifiedLedger: uint32(entry.LastModifiedLedgerSeq),
		Flags:              uint32(cBalance.Flags()),
	}
	return i.builder.RowStruct(ctx, row)
}

func (i *claimableBalancesBatchInsertBuilder) Exec(ctx context.Context) error {
	return i.builder.Exec(ctx)
}

var claimableBalancesSelectStatement = "cb.id, " +
	"cb.claimants, " +
	"cb.asset, " +
	"cb.amount, " +
	"cb.sponsor, " +
	"cb.last_modified_ledger, " +
	"cb.flags"

var selectClaimableBalances = sq.Select(claimableBalancesSelectStatement).From("claimable_balances cb")
