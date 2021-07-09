package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	strtime "github.com/stellar/go/support/time"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestTradeAggregations(t *testing.T) {
	itest := integration.NewTest(t, protocol15Config)
	ctx := context.Background()
	historyQ := itest.Horizon().HistoryQ()

	// Insert some trades
	now := strtime.Now().RoundDown(60_000)
	base, err := xdr.BuildAsset("credit_alphanum4", "GDUKMGUGDZQK6YHYA5Z6AY2G4XDSZPSZ3SW5UN3ARVMO6QSRDWP5YLEX", "EUR")
	counter, err := xdr.BuildAsset("credit_alphanum4", "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "USD")
	assert.NoError(t, err)

	assets, err := historyQ.CreateAssets(ctx, []xdr.Asset{
		base,
		counter,
	}, 1000)
	assert.NoError(t, err)
	assert.Len(t, assets, 2)
	baseAssetId := assets[base.String()].ID
	counterAssetId := assets[counter.String()].ID

	accounts, err := historyQ.CreateAccounts(ctx, []string{
		itest.Master().Address(),
	}, 1000)
	assert.NoError(t, err)
	assert.Len(t, accounts, 1)

	scenarios := []struct {
		name       string
		trades     []history.InsertTrade
		resolution int64
		offset     int64
		pq         db2.PageQuery
		startTime  strtime.Millis
		endTime    strtime.Millis
		expected   []history.TradeAggregation
	}{
		{
			name:       "no trades",
			trades:     []history.InsertTrade{},
			resolution: 60_000,
			pq:         db2.PageQuery{Limit: 100},
			expected:   []history.TradeAggregation{},
		},
		{
			name: "one trade",
			trades: []history.InsertTrade{
				{
					HistoryOperationID: 0,
					Order:              1,
					LedgerCloseTime:    now.ToTime().Add(5 * time.Second),
					SellerAccountID:    accounts[itest.Master().Address()],
					BuyerAccountID:     accounts[itest.Master().Address()],
					SoldAssetID:        baseAssetId,
					BoughtAssetID:      counterAssetId,
					Trade: xdr.ClaimOfferAtom{
						AssetSold:    base,
						AmountSold:   xdr.Int64(4_263_291_501),
						AssetBought:  counter,
						AmountBought: xdr.Int64(100),
					},
					SellPrice: xdr.Price{N: 23456, D: 10000},
				},
			},
			resolution: 60_000,
			pq:         db2.PageQuery{Limit: 100},
			expected: []history.TradeAggregation{
				{
					Timestamp:     now.ToInt64(),
					TradeCount:    1,
					BaseVolume:    "4263291501",
					CounterVolume: "100",
					Average:       float64(100) / 4_263_291_501,
					HighN:         23456,
					HighD:         10000,
					LowN:          23456,
					LowD:          10000,
					OpenN:         23456,
					OpenD:         10000,
					CloseN:        23456,
					CloseD:        10000,
				},
			},
		},
		{
			name: "two trades",
			trades: []history.InsertTrade{
				{
					HistoryOperationID: 0,
					Order:              0,
					LedgerCloseTime:    now.ToTime().Add(5 * time.Second),
					SellerAccountID:    accounts[itest.Master().Address()],
					BuyerAccountID:     accounts[itest.Master().Address()],
					SoldAssetID:        baseAssetId,
					BoughtAssetID:      counterAssetId,
					Trade: xdr.ClaimOfferAtom{
						AssetSold:    base,
						AmountSold:   xdr.Int64(4_263_291_501),
						AssetBought:  counter,
						AmountBought: xdr.Int64(100),
					},
					SellPrice: xdr.Price{N: 23456, D: 10000},
				},
				{
					HistoryOperationID: 0,
					Order:              1,
					LedgerCloseTime:    now.ToTime().Add(5 * time.Second),
					SellerAccountID:    accounts[itest.Master().Address()],
					BuyerAccountID:     accounts[itest.Master().Address()],
					SoldAssetID:        baseAssetId,
					BoughtAssetID:      counterAssetId,
					Trade: xdr.ClaimOfferAtom{
						AssetSold:    base,
						AmountSold:   xdr.Int64(4_263_291_501),
						AssetBought:  counter,
						AmountBought: xdr.Int64(1000),
					},
					SellPrice: xdr.Price{N: 13456, D: 10000},
				},
			},
			resolution: 60_000,
			pq:         db2.PageQuery{Limit: 100},
			expected: []history.TradeAggregation{
				{
					Timestamp:     now.ToInt64(),
					TradeCount:    2,
					BaseVolume:    "8526583002",
					CounterVolume: "1100",
					Average:       float64(1100) / 8_526_583_002,
					HighN:         23456,
					HighD:         10000,
					LowN:          13456,
					LowD:          10000,
					OpenN:         23456,
					OpenD:         10000,
					CloseN:        13456,
					CloseD:        10000,
				},
			},
		},
		{
			name: "1d resolution rollups",
			trades: []history.InsertTrade{
				{
					HistoryOperationID: 0,
					Order:              0,
					LedgerCloseTime:    now.ToTime().Add(5 * time.Second),
					SellerAccountID:    accounts[itest.Master().Address()],
					BuyerAccountID:     accounts[itest.Master().Address()],
					SoldAssetID:        baseAssetId,
					BoughtAssetID:      counterAssetId,
					Trade: xdr.ClaimOfferAtom{
						AssetSold:    base,
						AmountSold:   xdr.Int64(4_263_301_501),
						AssetBought:  counter,
						AmountBought: xdr.Int64(100),
					},
					SellPrice: xdr.Price{N: 23456, D: 10000},
				},
				{
					HistoryOperationID: 0,
					Order:              1,
					LedgerCloseTime:    now.ToTime().Add(5 * time.Second),
					SellerAccountID:    accounts[itest.Master().Address()],
					BuyerAccountID:     accounts[itest.Master().Address()],
					SoldAssetID:        baseAssetId,
					BoughtAssetID:      counterAssetId,
					Trade: xdr.ClaimOfferAtom{
						AssetSold:    base,
						AmountSold:   xdr.Int64(4_263_291_501),
						AssetBought:  counter,
						AmountBought: xdr.Int64(1000),
					},
					SellPrice: xdr.Price{N: 13456, D: 10000},
				},
			},
			resolution: 86_400_000,
			pq:         db2.PageQuery{Limit: 100},
			expected: []history.TradeAggregation{
				{
					Timestamp:     now.RoundDown(86_400_000).ToInt64(),
					TradeCount:    2,
					BaseVolume:    "8526593002",
					CounterVolume: "1100",
					Average:       float64(1100) / 8_526_593_002,
					HighN:         23456,
					HighD:         10000,
					LowN:          13456,
					LowD:          10000,
					OpenN:         23456,
					OpenD:         10000,
					CloseN:        13456,
					CloseD:        10000,
				},
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Run each in a txn so the ids don't conflict.
			assert.NoError(t, historyQ.Begin())
			defer func() {
				assert.NoError(t, historyQ.Rollback())
			}()

			batch := historyQ.NewTradeBatchInsertBuilder(1000)
			batch.Add(ctx, scenario.trades...)
			assert.NoError(t, batch.Exec(ctx))

			// Rebuild the aggregates.
			for _, trade := range scenario.trades {
				ledgerCloseTime := strtime.MillisFromTime(trade.LedgerCloseTime)
				assert.NoError(t, historyQ.RebuildTradeAggregationTimes(ctx, ledgerCloseTime, ledgerCloseTime))
			}

			// Check the result is what we expect
			query, err := historyQ.GetTradeAggregationsQ(
				baseAssetId,
				counterAssetId,
				scenario.resolution,
				scenario.offset,
				scenario.pq,
			)
			assert.NoError(t, err)

			if !scenario.startTime.IsNil() {
				query, err = query.WithStartTime(scenario.startTime)
				assert.NoError(t, err)
			}
			if !scenario.endTime.IsNil() {
				query, err = query.WithStartTime(scenario.endTime)
				assert.NoError(t, err)
			}

			sql, args, err := query.GetSql().ToSql()
			assert.NoError(t, err)
			t.Logf(
				"Querying sql: %q, args: %v",
				sql, args,
			)

			var records []history.TradeAggregation
			assert.NoError(t, historyQ.Select(ctx, &records, query.GetSql()))

			assert.Len(t, records, len(scenario.expected))
			for i, elem := range records {
				assert.Equal(t, scenario.expected[i], elem)
			}
		})
	}
}
