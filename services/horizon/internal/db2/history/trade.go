package history

import (
	"context"
	"fmt"
	"math"

	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// PagingToken returns a cursor for this trade
func (r *Trade) PagingToken() string {
	return fmt.Sprintf("%d-%d", r.HistoryOperationID, r.Order)
}

// HasPrice returns true if the trade has non-null price data
func (r *Trade) HasPrice() bool {
	return r.PriceN.Valid && r.PriceD.Valid
}

// Trades provides a helper to filter rows from the `history_trades` table
// with pre-defined filters.  See `TradesQ` methods for the available filters.
func (q *Q) Trades() *TradesQ {
	return &TradesQ{
		parent: q,
		sql: joinTradeAssets(
			joinTradeAccounts(
				selectTradeFields.From("history_trades htrd"),
				"history_accounts",
			),
			"history_assets",
		),
	}
}

// ReverseTrades provides a helper to filter rows from the `history_trades` table
// with pre-defined filters and reversed base/counter.  See `TradesQ` methods for the available filters.
func (q *Q) ReverseTrades() *TradesQ {
	return &TradesQ{
		parent: q,
		sql: joinTradeAssets(
			joinTradeAccounts(
				selectReverseTradeFields.From("history_trades htrd"),
				"history_accounts",
			),
			"history_assets",
		),
	}
}

// TradesForAssetPair provides a helper to filter rows from the `history_trades` table
// with the base filter of a specific asset pair.  See `TradesQ` methods for further available filters.
func (q *Q) TradesForAssetPair(baseAssetId int64, counterAssetId int64) *TradesQ {
	orderPreserved, baseAssetId, counterAssetId := getCanonicalAssetOrder(baseAssetId, counterAssetId)
	var trades *TradesQ
	if orderPreserved {
		trades = q.Trades()
	} else {
		trades = q.ReverseTrades()
	}
	return trades.forAssetPair(baseAssetId, counterAssetId)
}

// ForOffer filters the query results by the offer id.
func (q *TradesQ) ForOffer(id int64) *TradesQ {
	q.forOfferID = id
	return q
}

//Filter by asset pair. This function is private to ensure that correct order and proper select statement are coupled
func (q *TradesQ) forAssetPair(baseAssetId int64, counterAssetId int64) *TradesQ {
	q.sql = q.sql.Where(sq.Eq{"base_asset_id": baseAssetId, "counter_asset_id": counterAssetId})
	return q
}

// ForAccount filter Trades by account id
func (q *TradesQ) ForAccount(ctx context.Context, aid string) *TradesQ {
	var account Account
	q.Err = q.parent.AccountByAddress(ctx, &account, aid)
	if q.Err != nil {
		return q
	}

	q.forAccountID = account.ID
	return q
}

// Page specifies the paging constraints for the query being built by `q`.
func (q *TradesQ) Page(ctx context.Context, page db2.PageQuery) *TradesQ {
	if q.Err != nil {
		return q
	}

	op, idx, err := page.CursorInt64Pair(db2.DefaultPairSep)
	if err != nil {
		q.Err = err
		return q
	}

	// constrain the second portion of the cursor pair to 32-bits
	if idx > math.MaxInt32 {
		idx = math.MaxInt32
	}

	q.pageCalled = true

	if q.forAccountID != 0 || q.forOfferID != 0 {
		// Construct UNION query
		var firstSelect, secondSelect sq.SelectBuilder
		switch {
		case q.forAccountID != 0:
			firstSelect = q.sql.Where("htrd.base_account_id = ?", q.forAccountID)
			secondSelect = q.sql.Where("htrd.counter_account_id = ?", q.forAccountID)
		case q.forOfferID != 0:
			firstSelect = q.sql.Where("htrd.base_offer_id = ?", q.forOfferID)
			secondSelect = q.sql.Where("htrd.counter_offer_id = ?", q.forOfferID)
		}

		firstSelect = q.appendOrdering(ctx, firstSelect, op, idx, page.Order)
		secondSelect = q.appendOrdering(ctx, secondSelect, op, idx, page.Order)

		firstSQL, firstArgs, err := firstSelect.ToSql()
		if err != nil {
			q.Err = errors.New("error building a firstSelect query")
			return q
		}
		secondSQL, secondArgs, err := secondSelect.ToSql()
		if err != nil {
			q.Err = errors.New("error building a secondSelect query")
			return q
		}

		q.rawSQL = fmt.Sprintf("(%s) UNION (%s) ", firstSQL, secondSQL)
		q.rawArgs = append(q.rawArgs, firstArgs...)
		q.rawArgs = append(q.rawArgs, secondArgs...)
		// Order the final UNION:
		switch page.Order {
		case "asc":
			q.rawSQL = q.rawSQL + `ORDER BY history_operation_id asc, "order" asc `
		case "desc":
			q.rawSQL = q.rawSQL + `ORDER BY history_operation_id desc, "order" desc `
		default:
			panic("Invalid order")
		}
		q.rawSQL = q.rawSQL + fmt.Sprintf("LIMIT %d", page.Limit)
		// Reset sql so it's not used accidentally
		q.sql = sq.SelectBuilder{}
	} else {
		q.sql = q.appendOrdering(ctx, q.sql, op, idx, page.Order)
		q.sql = q.sql.Limit(page.Limit)
	}
	return q
}

func (q *TradesQ) appendOrdering(ctx context.Context, sel sq.SelectBuilder, op, idx int64, order string) sq.SelectBuilder {
	// NOTE: Remember to test the queries below with EXPLAIN / EXPLAIN ANALYZE
	// before changing them.
	// This condition is using multicolumn index and it's easy to write it in a way that
	// DB will perform a full table scan.
	switch order {
	case "asc":
		return sel.
			Where(`(
				htrd.history_operation_id >= ?
			AND (
				htrd.history_operation_id > ? OR
				(htrd.history_operation_id = ? AND htrd.order > ?)
			))`, op, op, op, idx).
			OrderBy("htrd.history_operation_id asc, htrd.order asc")
	case "desc":
		return sel.
			Where(`(
				htrd.history_operation_id <= ?
			AND (
				htrd.history_operation_id < ? OR
				(htrd.history_operation_id = ? AND htrd.order < ?)
			))`, op, op, op, idx).
			OrderBy("htrd.history_operation_id desc, htrd.order desc")
	default:
		panic("Invalid order")
	}
}

// Select loads the results of the query specified by `q` into `dest`.
func (q *TradesQ) Select(ctx context.Context, dest interface{}) error {
	if q.Err != nil {
		return q.Err
	}

	if !q.pageCalled {
		return errors.New("TradesQ.Page call is required before calling Select")
	}

	if q.rawSQL != "" {
		q.Err = q.parent.SelectRaw(ctx, dest, q.rawSQL, q.rawArgs...)
	} else {
		q.Err = q.parent.Select(ctx, dest, q.sql)
	}
	return q.Err
}

func joinTradeAccounts(selectBuilder sq.SelectBuilder, historyAccountsTable string) sq.SelectBuilder {
	return selectBuilder.
		Join(historyAccountsTable + " base_accounts ON base_account_id = base_accounts.id").
		Join(historyAccountsTable + " counter_accounts ON counter_account_id = counter_accounts.id")
}

func joinTradeAssets(selectBuilder sq.SelectBuilder, historyAssetsTable string) sq.SelectBuilder {
	return selectBuilder.
		Join(historyAssetsTable + " base_assets ON base_asset_id = base_assets.id").
		Join(historyAssetsTable + " counter_assets ON counter_asset_id = counter_assets.id")
}

var selectTradeFields = sq.Select(
	"history_operation_id",
	"htrd.\"order\"",
	"htrd.ledger_closed_at",
	"htrd.offer_id",
	"htrd.base_offer_id",
	"base_accounts.address as base_account",
	"base_assets.asset_type as base_asset_type",
	"base_assets.asset_code as base_asset_code",
	"base_assets.asset_issuer as base_asset_issuer",
	"htrd.base_amount",
	"htrd.counter_offer_id",
	"counter_accounts.address as counter_account",
	"counter_assets.asset_type as counter_asset_type",
	"counter_assets.asset_code as counter_asset_code",
	"counter_assets.asset_issuer as counter_asset_issuer",
	"htrd.counter_amount",
	"htrd.base_is_seller",
	"htrd.price_n",
	"htrd.price_d",
)

var selectReverseTradeFields = sq.Select(
	"history_operation_id",
	"htrd.\"order\"",
	"htrd.ledger_closed_at",
	"htrd.offer_id",
	"htrd.counter_offer_id as base_offer_id",
	"counter_accounts.address as base_account",
	"counter_assets.asset_type as base_asset_type",
	"counter_assets.asset_code as base_asset_code",
	"counter_assets.asset_issuer as base_asset_issuer",
	"htrd.counter_amount as base_amount",
	"htrd.base_offer_id as counter_offer_id",
	"base_accounts.address as counter_account",
	"base_assets.asset_type as counter_asset_type",
	"base_assets.asset_code as counter_asset_code",
	"base_assets.asset_issuer as counter_asset_issuer",
	"htrd.base_amount as counter_amount",
	"NOT(htrd.base_is_seller) as base_is_seller",
	"htrd.price_d as price_n",
	"htrd.price_n as price_d",
)

func getCanonicalAssetOrder(assetId1 int64, assetId2 int64) (orderPreserved bool, baseAssetId int64, counterAssetId int64) {
	if assetId1 < assetId2 {
		return true, assetId1, assetId2
	} else {
		return false, assetId2, assetId1
	}
}

type QTrades interface {
	QCreateAccountsHistory
	NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder
	CreateAssets(ctx context.Context, assets []xdr.Asset, maxBatchSize int) (map[string]Asset, error)
}
