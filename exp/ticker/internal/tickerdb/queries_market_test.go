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
	assert.Equal(t, 0.1, xlmbtcMkt.OpenPrice24h)
	assert.Equal(t, 0.1, xlmbtcMkt.LowestPrice24h)
	assert.Equal(t, 0.1, xlmbtcMkt.HighestPrice24h)

	assert.Equal(t, 150.0, xlmbtcMkt.BaseVolume7d)
	assert.Equal(t, 16.0, xlmbtcMkt.CounterVolume7d)
	assert.Equal(t, int64(2), xlmbtcMkt.TradeCount7d)
	assert.Equal(t, 0.12, xlmbtcMkt.OpenPrice7d)
	assert.Equal(t, 0.1, xlmbtcMkt.LowestPrice7d)
	assert.Equal(t, 0.12, xlmbtcMkt.HighestPrice7d)

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
	assert.Equal(t, 0.92, xlmethMkt.OpenPrice24h)
	assert.Equal(t, 0.92, xlmethMkt.LowestPrice24h)
	assert.Equal(t, 1.0, xlmethMkt.HighestPrice24h)

	assert.Equal(t, 74.0, xlmethMkt.BaseVolume7d)
	assert.Equal(t, 76.0, xlmethMkt.CounterVolume7d)
	assert.Equal(t, int64(2), xlmethMkt.TradeCount7d)
	assert.Equal(t, 0.92, xlmethMkt.OpenPrice7d)
	assert.Equal(t, 0.92, xlmethMkt.LowestPrice7d)
	assert.Equal(t, 1.0, xlmethMkt.HighestPrice7d)

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

func TestRetrievePartialMarkets(t *testing.T) {
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
	issuer1PK := "GCF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB"
	tbl := session.GetTable("issuers")
	_, err = tbl.Insert(Issuer{
		PublicKey: issuer1PK,
		Name:      "FOO BAR",
	}).IgnoreCols("id").Exec()
	require.NoError(t, err)
	var issuer1 Issuer
	err = session.GetRaw(&issuer1, `
		SELECT *
		FROM issuers
		WHERE public_key = ?`,
		issuer1PK,
	)
	require.NoError(t, err)

	// Adding another issuer to be used later:
	issuer2PK := "ABF3TQXKZJNFJK7HCMNE2O2CUNKCJH2Y2ROISTBPLC7C5EIA5NNG2XZB"
	_, err = tbl.Insert(Issuer{
		PublicKey: issuer2PK,
		Name:      "FOO BAR",
	}).IgnoreCols("id").Exec()
	require.NoError(t, err)
	var issuer2 Issuer
	err = session.GetRaw(&issuer2, `
		SELECT *
		FROM issuers
		WHERE public_key = ?`,
		issuer2PK,
	)
	require.NoError(t, err)

	// Adding a seed asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:          "ETH",
		IssuerAccount: issuer1PK,
		IssuerID:      issuer1.ID,
		IsValid:       true,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var ethAsset1 Asset
	err = session.GetRaw(&ethAsset1, `
		SELECT *
		FROM assets
		WHERE code = ?
		AND issuer_account = ?`,
		"ETH",
		issuer1PK,
	)
	require.NoError(t, err)

	// Adding a seed asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:          "ETH",
		IssuerAccount: issuer2PK,
		IssuerID:      issuer2.ID,
		IsValid:       true,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var ethAsset2 Asset
	err = session.GetRaw(&ethAsset2, `
		SELECT *
		FROM assets
		WHERE code = ?
		AND issuer_account = ?`,
		"ETH",
		issuer2PK,
	)
	require.NoError(t, err)

	// Adding another asset to be used later:
	err = session.InsertOrUpdateAsset(&Asset{
		Code:          "BTC",
		IssuerAccount: issuer1PK,
		IssuerID:      issuer1.ID,
		IsValid:       true,
	}, []string{"code", "issuer_id"})
	require.NoError(t, err)
	var btcAsset Asset
	err = session.GetRaw(&btcAsset, `
		SELECT *
		FROM assets
		WHERE code = ?
		AND issuer_account = ?`,
		"BTC",
		issuer1PK,
	)
	require.NoError(t, err)

	// A few times to be used:
	now := time.Now()
	tenMinutesAgo := now.Add(-10 * time.Minute)
	oneHourAgo := now.Add(-1 * time.Hour)
	threeDaysAgo := now.AddDate(0, 0, -3)

	// Now let's create the trades:
	trades := []Trade{
		Trade{ // BTC_ETH  trade (ETH is from issuer 1)
			HorizonID:       "hrzid1",
			BaseAssetID:     btcAsset.ID,
			BaseAmount:      100.0,
			CounterAssetID:  ethAsset1.ID,
			CounterAmount:   10.0,
			Price:           0.1,
			LedgerCloseTime: tenMinutesAgo,
		},
		Trade{ // BTC_ETH trade (ETH is from issuer 2)
			HorizonID:       "hrzid3",
			BaseAssetID:     btcAsset.ID,
			BaseAmount:      24.0,
			CounterAssetID:  ethAsset2.ID,
			CounterAmount:   26.0,
			Price:           0.92,
			LedgerCloseTime: now,
		},
		Trade{ // BTC_ETH  trade (ETH is from issuer 1)
			HorizonID:       "hrzid2",
			BaseAssetID:     btcAsset.ID,
			BaseAmount:      50.0,
			CounterAssetID:  ethAsset1.ID,
			CounterAmount:   50.0,
			Price:           1.0,
			LedgerCloseTime: oneHourAgo,
		},
		Trade{ // BTC_ETH  trade (ETH is from issuer 1)
			HorizonID:       "hrzid4",
			BaseAssetID:     btcAsset.ID,
			BaseAmount:      50.0,
			CounterAssetID:  ethAsset1.ID,
			CounterAmount:   6.0,
			Price:           0.12,
			LedgerCloseTime: threeDaysAgo,
		},
	}
	err = session.BulkInsertTrades(trades)
	require.NoError(t, err)

	partialMkts, err := session.RetrievePartialMarkets(
		nil, nil, nil, nil, 12,
	)
	require.NoError(t, err)
	assert.Equal(t, 2, len(partialMkts))

	// Mapping the retrieved markets:
	var btceth1Mkt, btceth2Mkt PartialMarket
	for _, mkt := range partialMkts {
		if mkt.CounterAssetIssuer == issuer1PK {
			btceth1Mkt = mkt
		}

		if mkt.CounterAssetIssuer == issuer2PK {
			btceth2Mkt = mkt
		}
	}
	tradePair1 := fmt.Sprintf("BTC:%s / ETH:%s", issuer1PK, issuer1PK)
	tradePair2 := fmt.Sprintf("BTC:%s / ETH:%s", issuer1PK, issuer2PK)

	assert.Equal(t, tradePair1, btceth1Mkt.TradePairName)
	assert.Equal(t, tradePair2, btceth2Mkt.TradePairName)

	// Validating the aggregated data
	assert.Equal(t, 150.0, btceth1Mkt.BaseVolume)
	assert.Equal(t, 60.0, btceth1Mkt.CounterVolume)
	assert.Equal(t, int32(2), btceth1Mkt.TradeCount)
	assert.Equal(t, 1.0, btceth1Mkt.Open)
	assert.Equal(t, 0.1, btceth1Mkt.Close)
	assert.Equal(t, -0.9, btceth1Mkt.Change)
	assert.Equal(t, 1.0, btceth1Mkt.High)
	assert.Equal(t, 0.1, btceth1Mkt.Low)
	assert.Equal(
		t,
		oneHourAgo.Local().Truncate(time.Millisecond),
		btceth1Mkt.FirstLedgerCloseTime.Local().Truncate(time.Millisecond),
	)

	assert.Equal(t, 24.0, btceth2Mkt.BaseVolume)
	assert.Equal(t, 26.0, btceth2Mkt.CounterVolume)
	assert.Equal(t, int32(1), btceth2Mkt.TradeCount)
	assert.Equal(t, 0.92, btceth2Mkt.Open)
	assert.Equal(t, 0.92, btceth2Mkt.Close)
	assert.Equal(t, 0.0, btceth2Mkt.Change)
	assert.Equal(t, 0.92, btceth2Mkt.High)
	assert.Equal(t, 0.92, btceth2Mkt.Low)
	assert.Equal(
		t,
		now.Local().Truncate(time.Millisecond),
		btceth2Mkt.FirstLedgerCloseTime.Local().Truncate(time.Millisecond),
	)

	// Now let's use the same data, but aggregating by asset pair
	partialAggMkts, err := session.RetrievePartialAggMarkets(nil, 12)
	require.NoError(t, err)
	assert.Equal(t, 1, len(partialAggMkts))

	partialAggMkt := partialAggMkts[0]

	assert.Equal(t, "BTC_ETH", partialAggMkt.TradePairName)
	assert.Equal(t, 174.0, partialAggMkt.BaseVolume)
	assert.Equal(t, 86.0, partialAggMkt.CounterVolume)
	assert.Equal(t, int32(3), partialAggMkt.TradeCount)
	assert.Equal(t, 1.0, partialAggMkt.Open)
	assert.Equal(t, 0.92, partialAggMkt.Close)
	assert.Equal(t, 1.0, partialAggMkt.High)
	assert.Equal(t, 0.1, partialAggMkt.Low)
	assert.Equal(
		t,
		oneHourAgo.Local().Truncate(time.Millisecond),
		partialAggMkt.FirstLedgerCloseTime.Local().Truncate(time.Millisecond),
	)

	// There might be some floating point rounding issues, so this test
	// needs to be a bit more flexible. Since the change is 0.08, an error
	// around 0.0000000000001 is acceptable:
	priceDiff := math.Abs(-0.08 - partialAggMkt.Change)
	assert.True(t, priceDiff < 0.0000000000001)

	// Validate the pair name parsing:
	pairName := new(string)
	*pairName = "BTC_ETH"
	partialAggMkts, err = session.RetrievePartialAggMarkets(pairName, 12)
	require.NoError(t, err)
	assert.Equal(t, 1, len(partialAggMkts))
	assert.Equal(t, int32(3), partialAggMkts[0].TradeCount)
}
