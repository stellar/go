package tickerdb

import (
	"fmt"
	"math"
	"testing"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/stellar/go/support/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrieveMarketData(t *testing.T) {
	db := dbtest.Postgres(t)
	defer db.Close()

	var session TickerSession
	session.DB = db.Open()
	defer session.DB.Close()

	// Run migrations to make sure the tests are run
	// on the most updated schema version
	migrations := &migrate.FileMigrationSource{
		Dir: "./migrations",
	}
	_, err := migrate.Exec(session.DB.DB, "postgres", migrations, migrate.Up)
	require.NoError(t, err)

	// Adding a seed issuer to be used later:
	tbl := session.GetTable("issuers")
	_, err = tbl.Insert(Issuer{
		PublicKey: "GCF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB",
		Name:      "FOO BAR",
	}).IgnoreCols("id").Exec()
	require.NoError(t, err)
	var issuer Issuer
	err = session.GetRaw(&issuer, `
		SELECT *
		FROM issuers
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Adding a seed asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:     "XLM",
		IssuerID: issuer.ID,
		IsValid:  true,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var xlmAsset Asset
	err = session.GetRaw(&xlmAsset, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Adding another asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:     "BTC",
		IssuerID: issuer.ID,
		IsValid:  true,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var btcAsset Asset
	err = session.GetRaw(&btcAsset, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Adding a third asset:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:     "ETH",
		IssuerID: issuer.ID,
		IsValid:  true,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var ethAsset Asset
	err = session.GetRaw(&ethAsset, `
		SELECT *
		FROM assets
		ORDER BY id DESC
		LIMIT 1`,
	)
	require.NoError(t, err)

	// Verify that we actually have three assets:
	assert.NotEqual(t, xlmAsset.ID, btcAsset.ID)
	assert.NotEqual(t, btcAsset.ID, ethAsset.ID)
	assert.NotEqual(t, xlmAsset.ID, ethAsset.ID)

	// A few times to be used:
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	threeDaysAgo := now.AddDate(0, 0, -3)
	oneMonthAgo := now.AddDate(0, -1, 0)

	// Now let's create the trades:
	trades := []Trade{
		Trade{ // XLM_BTC trade
			HorizonID:       "hrzid1",
			BaseAssetID:     xlmAsset.ID,
			BaseAmount:      100.0,
			CounterAssetID:  btcAsset.ID,
			CounterAmount:   10.0,
			Price:           0.1,
			LedgerCloseTime: now,
		},
		Trade{ // XLM_ETH trade
			HorizonID:       "hrzid3",
			BaseAssetID:     xlmAsset.ID,
			BaseAmount:      24.0,
			CounterAssetID:  ethAsset.ID,
			CounterAmount:   26.0,
			Price:           0.92,
			LedgerCloseTime: oneHourAgo,
		},
		Trade{ // XLM_ETH trade
			HorizonID:       "hrzid2",
			BaseAssetID:     xlmAsset.ID,
			BaseAmount:      50.0,
			CounterAssetID:  ethAsset.ID,
			CounterAmount:   50.0,
			Price:           1.0,
			LedgerCloseTime: now,
		},
		Trade{ // XLM_BTC trade
			HorizonID:       "hrzid4",
			BaseAssetID:     xlmAsset.ID,
			BaseAmount:      50.0,
			CounterAssetID:  btcAsset.ID,
			CounterAmount:   6.0,
			Price:           0.12,
			LedgerCloseTime: threeDaysAgo,
		},
		Trade{ // XLM_ETH trade
			HorizonID:       "hrzid5",
			BaseAssetID:     xlmAsset.ID,
			BaseAmount:      24.0,
			CounterAssetID:  ethAsset.ID,
			CounterAmount:   28.0,
			Price:           1.10,
			LedgerCloseTime: oneMonthAgo,
		},
	}
	err = session.BulkInsertTrades(trades)
	require.NoError(t, err)

	markets, err := session.RetrieveMarketData()
	require.NoError(t, err)
	fmt.Println(markets)
	assert.Equal(t, 2, len(markets))

	// Mapping the retrieved markets:
	var xlmbtcMkt, xlmethMkt Market
	for _, mkt := range markets {
		if mkt.TradePair == "XLM_BTC" {
			xlmbtcMkt = mkt
		}

		if mkt.TradePair == "XLM_ETH" {
			xlmethMkt = mkt
		}
	}
	assert.NotEqual(t, "", xlmbtcMkt.TradePair)
	assert.NotEqual(t, "", xlmethMkt.TradePair)

	// Validating the aggregated data
	assert.Equal(t, 100.0, xlmbtcMkt.BaseVolume24h)
	assert.Equal(t, 10.0, xlmbtcMkt.CounterVolume24h)
	assert.Equal(t, int64(1), xlmbtcMkt.TradeCount24h)

	assert.Equal(t, 150.0, xlmbtcMkt.BaseVolume7d)
	assert.Equal(t, 16.0, xlmbtcMkt.CounterVolume7d)
	assert.Equal(t, int64(2), xlmbtcMkt.TradeCount7d)

	assert.Equal(t, 0.1, xlmbtcMkt.LastPrice)
	assert.Equal(
		t,
		now.Local().Truncate(time.Millisecond),
		xlmbtcMkt.LastPriceCloseTime.Local().Truncate(time.Millisecond),
	)

	assert.Equal(t, 0.0, xlmbtcMkt.PriceChange24h)
	// There might be some floating point rounding issues, so this test
	// needs to be a bit more flexible. Since the change is 0.02, an error
	// around 0.0000000000001 is acceptable:
	priceChange7dDiff := math.Abs(0.02 - xlmbtcMkt.PriceChange7d)
	assert.True(t, priceChange7dDiff < 0.0000000000001)

	assert.Equal(t, 74.0, xlmethMkt.BaseVolume24h)
	assert.Equal(t, 76.0, xlmethMkt.CounterVolume24h)
	assert.Equal(t, int64(2), xlmethMkt.TradeCount24h)

	assert.Equal(t, 74.0, xlmethMkt.BaseVolume7d)
	assert.Equal(t, 76.0, xlmethMkt.CounterVolume7d)
	assert.Equal(t, int64(2), xlmethMkt.TradeCount7d)

	assert.Equal(t, 1.0, xlmethMkt.LastPrice)
	assert.Equal(
		t,
		now.Local().Truncate(time.Millisecond),
		xlmbtcMkt.LastPriceCloseTime.Local().Truncate(time.Millisecond),
	)

	// There might be some floating point rounding issues, so this test
	// needs to be a bit more flexible. Since the change is 0.08, an error
	// around 0.0000000000001 is acceptable:
	priceChange24hDiff := math.Abs(-0.08 - xlmethMkt.PriceChange24h)
	assert.True(t, priceChange24hDiff < 0.0000000000001)

	priceChange7dDiff = math.Abs(-0.08 - xlmethMkt.PriceChange7d)
	assert.True(t, priceChange7dDiff < 0.0000000000001)

	assert.Equal(t, priceChange24hDiff, priceChange7dDiff)
}
