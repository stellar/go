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
	native := xdr.MustNewNativeAsset()
	counter, err := xdr.BuildAsset("credit_alphanum4", "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "USD")
	assert.NoError(t, err)
	scenarios := []struct {
		name           string
		baseAssetId    int64
		counterAssetId int64
		trades         []history.InsertTrade
		resolution     int64
		offset         int64
		pq             db2.PageQuery
		startTime      strtime.Millis
		endTime        strtime.Millis
		expected       []history.TradeAggregation
	}{
		{
			name:           "no trades",
			baseAssetId:    2,
			counterAssetId: 149,
			trades:         []history.InsertTrade{},
			resolution:     60_000,
			startTime:      now,
			endTime:        now + 60_000,
			expected:       []history.TradeAggregation{},
		},
		{
			name:           "one trade",
			baseAssetId:    2,
			counterAssetId: 149,
			trades: []history.InsertTrade{
				{
					HistoryOperationID: 0,
					Order:              1,
					LedgerCloseTime:    now.ToTime().Add(5 * time.Second),
					SoldAssetID:        2,
					BoughtAssetID:      149,
					Trade: xdr.ClaimOfferAtom{
						AssetSold:    native,
						AmountSold:   xdr.Int64(4263301501),
						AssetBought:  counter,
						AmountBought: xdr.Int64(100),
					},
					SellPrice: xdr.Price{N: 23456, D: 10000},
				},
			},
			resolution: 60_000,
			startTime:  now,
			endTime:    now + 60_000,
			expected: []history.TradeAggregation{
				{
					Timestamp:     now.ToInt64(),
					TradeCount:    1,
					BaseVolume:    "426.3301501",
					CounterVolume: "0.0000100",
					Average:       float64(100) / 4263301501,
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
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			batch := historyQ.NewTradeBatchInsertBuilder(1000)
			batch.Add(ctx, scenario.trades...)
			assert.NoError(t, batch.Exec(ctx))

			// Check the result is what we expect
			query, err := historyQ.GetTradeAggregationsQ(
				scenario.baseAssetId,
				scenario.counterAssetId,
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

			var records []history.TradeAggregation
			assert.NoError(t, historyQ.Select(ctx, &records, query.GetSql()))

			assert.Len(t, records, len(scenario.expected))
			for i, elem := range records {
				assert.Equal(t, scenario.expected[i], elem)
			}
		})
	}
}
