package history

import (
	"github.com/stellar/go/xdr"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
)

var (
	ascPQ  = db2.MustPageQuery("", false, "asc", 100)
	descPQ = db2.MustPageQuery("", false, "desc", 100)
)

func assertTradesAreEqual(tt *test.T, expected, rows []Trade) {
	tt.Assert.Len(rows, len(expected))
	for i := 0; i < len(rows); i++ {
		tt.Assert.Equal(expected[i].LedgerCloseTime.Unix(), rows[i].LedgerCloseTime.Unix())
		rows[i].LedgerCloseTime = expected[i].LedgerCloseTime
		tt.Assert.Equal(
			expected[i],
			rows[i],
		)
	}
}

func TestSelectTrades(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	fixtures := TradeScenario(tt, q)

	rows, err := q.Trades().Page(tt.Ctx, ascPQ).Select(tt.Ctx)
	tt.Assert.NoError(err)

	assertTradesAreEqual(tt, fixtures.Trades, rows)

	rows, err = q.Trades().Page(tt.Ctx, descPQ).Select(tt.Ctx)
	tt.Assert.NoError(err)
	start, end := 0, len(rows)-1
	for start < end {
		rows[start], rows[end] = rows[end], rows[start]
		start++
		end--
	}

	assertTradesAreEqual(tt, fixtures.Trades, rows)
}

func TestSelectTradesCursor(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	fixtures := TradeScenario(tt, q)

	rows, err := q.Trades().Page(tt.Ctx, db2.MustPageQuery(fixtures.Trades[0].PagingToken(), false, "asc", 100)).Select(tt.Ctx)
	tt.Assert.NoError(err)
	assertTradesAreEqual(tt, rows, fixtures.Trades[1:])

	rows, err = q.Trades().Page(tt.Ctx, db2.MustPageQuery(fixtures.Trades[1].PagingToken(), false, "asc", 100)).Select(tt.Ctx)
	tt.Assert.NoError(err)
	assertTradesAreEqual(tt, rows, fixtures.Trades[2:])
}

func TestTradesQueryForAccount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	fixtures := TradeScenario(tt, q)
	tt.Assert.NotEmpty(fixtures.TradesByAccount)

	for account, expected := range fixtures.TradesByAccount {
		trades, err := q.Trades().ForAccount(tt.Ctx, account).Page(tt.Ctx, ascPQ).Select(tt.Ctx)
		tt.Assert.NoError(err)
		assertTradesAreEqual(tt, expected, trades)
		trades, err = q.Trades().ForAccount(tt.Ctx, account).Page(tt.Ctx, db2.MustPageQuery(expected[0].PagingToken(), false, "asc", 100)).Select(tt.Ctx)
		tt.Assert.NoError(err)
		assertTradesAreEqual(tt, expected[1:], trades)
	}
}

func TestTradesQueryForOffer(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	fixtures := TradeScenario(tt, q)
	tt.Assert.NotEmpty(fixtures.TradesByOffer)

	for offer, expected := range fixtures.TradesByOffer {
		trades, err := q.Trades().ForOffer(offer).Page(tt.Ctx, ascPQ).Select(tt.Ctx)
		tt.Assert.NoError(err)
		assertTradesAreEqual(tt, expected, trades)
		trades, err = q.Trades().ForOffer(offer).Page(tt.Ctx, db2.MustPageQuery(expected[0].PagingToken(), false, "asc", 100)).Select(tt.Ctx)
		tt.Assert.NoError(err)
		assertTradesAreEqual(tt, expected[1:], trades)
	}
}

func TestTradesForAssetPair(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	fixtures := TradeScenario(tt, q)

	eurAsset := xdr.MustNewCreditAsset("EUR", issuer.Address())
	chfAsset := xdr.MustNewCreditAsset("CHF", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU")

	expected := fixtures.TradesByAssetPair(eurAsset, chfAsset)
	tt.Assert.NotEmpty(expected)

	eurAssetID, err := q.GetAssetID(tt.Ctx, eurAsset)
	tt.Assert.NoError(err)

	chfAssetID, err := q.GetAssetID(tt.Ctx, chfAsset)
	tt.Assert.NoError(err)

	trades, err := q.TradesForAssetPair(chfAssetID, eurAssetID).Page(tt.Ctx, ascPQ).Select(tt.Ctx)
	tt.Assert.NoError(err)

	assertTradesAreEqual(tt, expected, trades)

	trades, err = q.TradesForAssetPair(chfAssetID, eurAssetID).Page(tt.Ctx, db2.MustPageQuery(expected[0].PagingToken(), false, "asc", 100)).Select(tt.Ctx)
	tt.Assert.NoError(err)
	assertTradesAreEqual(tt, expected[1:], trades)
}

func reverseTrade(expected Trade) Trade {
	expected.BaseIsSeller = !expected.BaseIsSeller
	expected.BaseAssetCode, expected.CounterAssetCode = expected.CounterAssetCode, expected.BaseAssetCode
	expected.BaseAssetIssuer, expected.CounterAssetIssuer = expected.CounterAssetIssuer, expected.BaseAssetIssuer
	expected.BaseOfferID, expected.CounterOfferID = expected.CounterOfferID, expected.BaseOfferID
	expected.BaseLiquidityPoolID, expected.CounterLiquidityPoolID = expected.CounterLiquidityPoolID, expected.BaseLiquidityPoolID
	expected.BaseAssetType, expected.CounterAssetType = expected.CounterAssetType, expected.BaseAssetType
	expected.BaseAccount, expected.CounterAccount = expected.CounterAccount, expected.BaseAccount
	expected.BaseAmount, expected.CounterAmount = expected.CounterAmount, expected.BaseAmount
	expected.PriceN, expected.PriceD = expected.PriceD, expected.PriceN
	return expected
}

func TestTradesForReverseAssetPair(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	fixtures := TradeScenario(tt, q)

	eurAsset := xdr.MustNewCreditAsset("EUR", issuer.Address())
	chfAsset := xdr.MustNewCreditAsset("CHF", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU")

	expected := fixtures.TradesByAssetPair(eurAsset, chfAsset)
	tt.Assert.NotEmpty(expected)
	for i := range expected {
		expected[i] = reverseTrade(expected[i])
	}

	eurAssetID, err := q.GetAssetID(tt.Ctx, eurAsset)
	tt.Assert.NoError(err)

	chfAssetID, err := q.GetAssetID(tt.Ctx, chfAsset)
	tt.Assert.NoError(err)

	trades, err := q.TradesForAssetPair(eurAssetID, chfAssetID).Page(tt.Ctx, ascPQ).Select(tt.Ctx)
	tt.Assert.NoError(err)

	assertTradesAreEqual(tt, expected, trades)

	trades, err = q.TradesForAssetPair(eurAssetID, chfAssetID).Page(tt.Ctx, db2.MustPageQuery(expected[0].PagingToken(), false, "asc", 100)).Select(tt.Ctx)
	tt.Assert.NoError(err)
	assertTradesAreEqual(tt, expected[1:], trades)
}
