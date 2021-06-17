package history

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	strtime "github.com/stellar/go/support/time"
	"github.com/stellar/go/xdr"
)

// InsertTrade represents the arguments to TradeBatchInsertBuilder.Add() which is used to insert
// rows into the history_trades table
type InsertTrade struct {
	HistoryOperationID int64
	Order              int32
	LedgerCloseTime    time.Time
	BuyOfferExists     bool
	BuyOfferID         int64
	SellerAccountID    int64
	BuyerAccountID     int64
	SoldAssetID        int64
	BoughtAssetID      int64
	Trade              xdr.ClaimOfferAtom
	SellPrice          xdr.Price
}

// TradeBatchInsertBuilder is used to insert trades into the
// history_trades table
type TradeBatchInsertBuilder interface {
	Add(ctx context.Context, entries ...InsertTrade) error
	Exec(ctx context.Context) error
}

// tradeBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type tradeBatchInsertBuilder struct {
	builder        db.BatchInsertBuilder
	q              *Q
	updatedBuckets map[int64]struct{}
}

// NewTradeBatchInsertBuilder constructs a new TradeBatchInsertBuilder instance
func (q *Q) NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder {
	return &tradeBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_trades"),
			MaxBatchSize: maxBatchSize,
		},
		q:              q,
		updatedBuckets: map[int64]struct{}{},
	}
}

// Exec flushes all outstanding trades to the database
func (i *tradeBatchInsertBuilder) Exec(ctx context.Context) error {
	err := i.builder.Exec(ctx)
	if err != nil {
		return err
	}

	// Rebuild all updated buckets.
	var buckets []int64
	for bucket := range i.updatedBuckets {
		buckets = append(buckets, bucket)
	}
	return i.rebuildTradeAggregationBuckets(ctx, buckets)
}

// rebuildTradeAggregationBuckets rebuilds a specific set of trade aggregation buckets.
// to ensure complete data in case of partial reingestion.
func (i *tradeBatchInsertBuilder) rebuildTradeAggregationBuckets(ctx context.Context, buckets []int64) error {
	// Clear out the old bucket values.
	_, err := i.q.Exec(ctx, sq.Delete("history_trades_60000").Where(sq.Eq{
		"timestamp": buckets,
	}))
	if err != nil {
		return errors.Wrap(err, "could rebuild trade aggregation bucket")
	}

	// find all related trades
	trades := sq.Select(
		"public.to_millis(ledger_closed_at, 60000) as timestamp",
		"history_operation_id",
		"order",
		"base_asset_id",
		"base_amount",
		"counter_asset_id",
		"counter_amount",
		"ARRAY[price_n, price_d] as price",
	).From("history_trades").Where(sq.Eq{
		"to_millis(ledger_closed_at, 60000)": buckets,
	}).OrderBy("history_operation_id", "order")

	// figure out the new bucket values
	rebuilt := sq.Select(
		"timestamp",
		"base_asset_id",
		"counter_asset_id",
		"count(*) as count",
		"sum(base_amount) as base_volume",
		"sum(counter_amount) as counter_volume",
		"sum(counter_amount::numeric)/sum(base_amount::numeric) as avg",
		"(max_price(price))[1] as high_n",
		"(max_price(price))[2] as high_d",
		"(min_price(price))[1] as low_n",
		"(min_price(price))[2] as low_d",
		"first(history_operation_id) as open_ledger_seq",
		"(first(price))[1] as open_n",
		"(first(price))[2] as open_d",
		"last(history_operation_id) as close_ledger_seq",
		"(last(price))[1] as close_n",
		"(last(price))[2] as close_d",
	).FromSelect(trades, "trades").Where(sq.Eq{
		"timestamp": buckets,
	}).GroupBy("timestamp", "base_asset_id", "counter_asset_id")

	// Insert the new bucket values.
	// TODO: Upgrade squirrel to do this? or figure out another way...
	_, err = i.q.Exec(ctx, sq.Insert("history_trades_60000").Select(rebuilt))
	if err != nil {
		return errors.Wrap(err, "could rebuild trade aggregation bucket")
	}
	return nil
}

// Add adds a new trade to the batch
func (i *tradeBatchInsertBuilder) Add(ctx context.Context, entries ...InsertTrade) error {
	for _, entry := range entries {
		i.updatedBuckets[strtime.MillisFromTime(entry.LedgerCloseTime).RoundDown(60_000).ToInt64()] = struct{}{}

		sellOfferID := EncodeOfferId(uint64(entry.Trade.OfferId), CoreOfferIDType)

		// if the buy offer exists, encode the stellar core generated id as the offer id
		// if not, encode the toid as the offer id
		var buyOfferID int64
		if entry.BuyOfferExists {
			buyOfferID = EncodeOfferId(uint64(entry.BuyOfferID), CoreOfferIDType)
		} else {
			buyOfferID = EncodeOfferId(uint64(entry.HistoryOperationID), TOIDType)
		}

		orderPreserved, baseAssetID, counterAssetID := getCanonicalAssetOrder(
			entry.SoldAssetID, entry.BoughtAssetID,
		)

		var baseAccountID, counterAccountID int64
		var baseAmount, counterAmount xdr.Int64
		var baseOfferID, counterOfferID int64

		if orderPreserved {
			baseAccountID = entry.SellerAccountID
			baseAmount = entry.Trade.AmountSold
			counterAccountID = entry.BuyerAccountID
			counterAmount = entry.Trade.AmountBought
			baseOfferID = sellOfferID
			counterOfferID = buyOfferID
		} else {
			baseAccountID = entry.BuyerAccountID
			baseAmount = entry.Trade.AmountBought
			counterAccountID = entry.SellerAccountID
			counterAmount = entry.Trade.AmountSold
			baseOfferID = buyOfferID
			counterOfferID = sellOfferID
			entry.SellPrice.Invert()
		}

		err := i.builder.Row(ctx, map[string]interface{}{
			"history_operation_id": entry.HistoryOperationID,
			"\"order\"":            entry.Order,
			"ledger_closed_at":     entry.LedgerCloseTime,
			"offer_id":             entry.Trade.OfferId,
			"base_offer_id":        baseOfferID,
			"base_account_id":      baseAccountID,
			"base_asset_id":        baseAssetID,
			"base_amount":          baseAmount,
			"counter_offer_id":     counterOfferID,
			"counter_account_id":   counterAccountID,
			"counter_asset_id":     counterAssetID,
			"counter_amount":       counterAmount,
			"base_is_seller":       orderPreserved,
			"price_n":              entry.SellPrice.N,
			"price_d":              entry.SellPrice.D,
		})
		if err != nil {
			return errors.Wrap(err, "failed to add trade")
		}
	}

	return nil
}
