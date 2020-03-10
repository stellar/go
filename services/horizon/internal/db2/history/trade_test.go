package history

import (
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
	supportTime "github.com/stellar/go/support/time"
	"github.com/stellar/go/xdr"
)

func TestTradeQueries(t *testing.T) {
	tt := test.Start(t).Scenario("kahuna")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}
	var trades []Trade

	// All trades
	err := q.Trades().Page(db2.MustPageQuery("", false, "asc", 100)).Select(&trades)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(trades, 4)
	}

	// Paging
	pq := db2.MustPageQuery(trades[0].PagingToken(), false, "asc", 1)
	var pt []Trade

	err = q.Trades().Page(pq).Select(&pt)
	if tt.Assert.NoError(err) {
		if tt.Assert.Len(pt, 1) {
			tt.Assert.Equal(trades[1], pt[0])
		}
	}

	// Cursor bounds checking
	pq = db2.MustPageQuery("", false, "desc", 1)
	err = q.Trades().Page(pq).Select(&pt)
	tt.Require.NoError(err)

	// test for asset pairs
	lumen, err := q.GetAssetID(xdr.MustNewNativeAsset())
	tt.Require.NoError(err)
	assetUSD, err := q.GetAssetID(xdr.MustNewCreditAsset("USD", "GB2QIYT2IAUFMRXKLSLLPRECC6OCOGJMADSPTRK7TGNT2SFR2YGWDARD"))
	tt.Require.NoError(err)
	assetEUR, err := q.GetAssetID(xdr.MustNewCreditAsset("EUR", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU"))
	tt.Require.NoError(err)

	err = q.TradesForAssetPair(assetUSD, assetEUR).Select(&trades)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 0)

	assetUSD, err = q.GetAssetID(xdr.MustNewCreditAsset("USD", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU"))
	tt.Require.NoError(err)

	err = q.TradesForAssetPair(lumen, assetUSD).Select(&trades)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 1)

	tt.Assert.Equal(xdr.Int64(2000000000), trades[0].BaseAmount)
	tt.Assert.Equal(xdr.Int64(1000000000), trades[0].CounterAmount)
	tt.Assert.Equal(true, trades[0].BaseIsSeller)

	// reverse assets
	err = q.TradesForAssetPair(assetUSD, lumen).Select(&trades)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 1)

	tt.Assert.Equal(xdr.Int64(1000000000), trades[0].BaseAmount)
	tt.Assert.Equal(xdr.Int64(2000000000), trades[0].CounterAmount)
	tt.Assert.Equal(false, trades[0].BaseIsSeller)
}

func createInsertTrades(
	accountIDs []int64, assetIDs []int64, ledger int32,
) (InsertTrade, InsertTrade, InsertTrade) {
	first := InsertTrade{
		HistoryOperationID: toid.New(ledger, 1, 1).ToInt64(),
		Order:              1,
		LedgerCloseTime:    supportTime.MillisFromSeconds(time.Now().Unix()).ToTime(),
		BuyOfferExists:     true,
		BuyOfferID:         32145,
		SellerAccountID:    accountIDs[0],
		BuyerAccountID:     accountIDs[1],
		SoldAssetID:        assetIDs[0],
		BoughtAssetID:      assetIDs[1],
		SellPrice: xdr.Price{
			N: 1,
			D: 3,
		},
		Trade: xdr.ClaimOfferAtom{
			OfferId:      214515,
			AmountSold:   7986,
			AmountBought: 896,
		},
	}

	second := first
	second.BuyOfferExists = false
	second.BuyOfferID = 89
	second.Order = 2

	third := InsertTrade{
		HistoryOperationID: toid.New(ledger, 2, 1).ToInt64(),
		Order:              1,
		LedgerCloseTime:    time.Now().UTC(),
		BuyOfferExists:     true,
		BuyOfferID:         2,
		SellerAccountID:    accountIDs[1],
		BuyerAccountID:     accountIDs[0],
		SoldAssetID:        assetIDs[2],
		BoughtAssetID:      assetIDs[1],
		SellPrice: xdr.Price{
			N: 1156,
			D: 3,
		},
		Trade: xdr.ClaimOfferAtom{
			OfferId:      7,
			AmountSold:   123,
			AmountBought: 6,
		},
	}

	return first, second, third
}

func createAccountsAndAssets(
	tt *test.T, q *Q, accounts []string, assets []xdr.Asset,
) ([]int64, []int64) {
	addressToAccounts, err := q.CreateAccounts(accounts, 2)
	tt.Assert.NoError(err)

	accountIDs := []int64{}
	for _, account := range accounts {
		accountIDs = append(accountIDs, addressToAccounts[account])
	}

	assetMap, err := q.CreateAssets(assets, 2)
	tt.Assert.NoError(err)

	assetIDs := []int64{}
	for _, asset := range assets {
		assetIDs = append(assetIDs, assetMap[asset.String()].ID)
	}

	return accountIDs, assetIDs
}

func newInt64(v int64) *int64 {
	p := new(int64)
	*p = v
	return p
}

func buildIDtoAccountMapping(addresses []string, ids []int64) map[int64]xdr.AccountId {
	idToAccount := map[int64]xdr.AccountId{}
	for i, id := range ids {
		account := xdr.MustAddress(addresses[i])
		idToAccount[id] = account
	}

	return idToAccount
}

func buildIDtoAssetMapping(assets []xdr.Asset, ids []int64) map[int64]xdr.Asset {
	idToAsset := map[int64]xdr.Asset{}
	for i, id := range ids {
		idToAsset[id] = assets[i]
	}

	return idToAsset
}

func TestBatchInsertTrade(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	addresses := []string{
		"GB2QIYT2IAUFMRXKLSLLPRECC6OCOGJMADSPTRK7TGNT2SFR2YGWDARD",
		"GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU",
	}
	assets := []xdr.Asset{eurAsset, usdAsset, nativeAsset}
	accountIDs, assetIDs := createAccountsAndAssets(
		tt, q,
		addresses,
		assets,
	)

	first, second, third := createInsertTrades(accountIDs, assetIDs, 3)

	builder := q.NewTradeBatchInsertBuilder(1)
	tt.Assert.NoError(
		builder.Add(first, second, third),
	)
	tt.Assert.NoError(builder.Exec())

	var rows []Trade
	tt.Assert.NoError(q.Trades().Select(&rows))

	idToAccount := buildIDtoAccountMapping(addresses, accountIDs)
	idToAsset := buildIDtoAssetMapping(assets, assetIDs)

	firstSellerAccount := idToAccount[first.SellerAccountID]
	firstBuyerAccount := idToAccount[first.BuyerAccountID]
	var firstSoldAssetType, firstSoldAssetCode, firstSoldAssetIssuer string
	idToAsset[first.SoldAssetID].MustExtract(
		&firstSoldAssetType, &firstSoldAssetCode, &firstSoldAssetIssuer,
	)
	var firstBoughtAssetType, firstBoughtAssetCode, firstBoughtAssetIssuer string
	idToAsset[first.BoughtAssetID].MustExtract(
		&firstBoughtAssetType, &firstBoughtAssetCode, &firstBoughtAssetIssuer,
	)

	secondSellerAccount := idToAccount[second.SellerAccountID]
	secondBuyerAccount := idToAccount[second.BuyerAccountID]
	var secondSoldAssetType, secondSoldAssetCode, secondSoldAssetIssuer string
	idToAsset[second.SoldAssetID].MustExtract(
		&secondSoldAssetType, &secondSoldAssetCode, &secondSoldAssetIssuer,
	)
	var secondBoughtAssetType, secondBoughtAssetCode, secondBoughtAssetIssuer string
	idToAsset[second.BoughtAssetID].MustExtract(
		&secondBoughtAssetType, &secondBoughtAssetCode, &secondBoughtAssetIssuer,
	)

	thirdSellerAccount := idToAccount[third.SellerAccountID]
	thirdBuyerAccount := idToAccount[third.BuyerAccountID]
	var thirdSoldAssetType, thirdSoldAssetCode, thirdSoldAssetIssuer string
	idToAsset[third.SoldAssetID].MustExtract(
		&thirdSoldAssetType, &thirdSoldAssetCode, &thirdSoldAssetIssuer,
	)
	var thirdBoughtAssetType, thirdBoughtAssetCode, thirdBoughtAssetIssuer string
	idToAsset[third.BoughtAssetID].MustExtract(
		&thirdBoughtAssetType, &thirdBoughtAssetCode, &thirdBoughtAssetIssuer,
	)

	expected := []Trade{
		Trade{
			HistoryOperationID: first.HistoryOperationID,
			Order:              first.Order,
			LedgerCloseTime:    first.LedgerCloseTime,
			OfferID:            int64(first.Trade.OfferId),
			BaseOfferID:        newInt64(EncodeOfferId(uint64(first.Trade.OfferId), CoreOfferIDType)),
			BaseAccount:        firstSellerAccount.Address(),
			BaseAssetType:      firstSoldAssetType,
			BaseAssetIssuer:    firstSoldAssetIssuer,
			BaseAssetCode:      firstSoldAssetCode,
			BaseAmount:         first.Trade.AmountSold,
			CounterOfferID:     newInt64(first.BuyOfferID),
			CounterAccount:     firstBuyerAccount.Address(),
			CounterAssetType:   firstBoughtAssetType,
			CounterAssetIssuer: firstBoughtAssetIssuer,
			CounterAssetCode:   firstBoughtAssetCode,
			CounterAmount:      first.Trade.AmountBought,
			BaseIsSeller:       true,
			PriceN:             null.NewInt(int64(first.SellPrice.N), true),
			PriceD:             null.NewInt(int64(first.SellPrice.D), true),
		},
		Trade{
			HistoryOperationID: second.HistoryOperationID,
			Order:              second.Order,
			LedgerCloseTime:    second.LedgerCloseTime,
			OfferID:            int64(second.Trade.OfferId),
			BaseOfferID:        newInt64(EncodeOfferId(uint64(second.Trade.OfferId), CoreOfferIDType)),
			BaseAccount:        secondSellerAccount.Address(),
			BaseAssetType:      secondSoldAssetType,
			BaseAssetIssuer:    secondSoldAssetIssuer,
			BaseAssetCode:      secondSoldAssetCode,
			BaseAmount:         second.Trade.AmountSold,
			CounterOfferID:     newInt64(EncodeOfferId(uint64(second.HistoryOperationID), TOIDType)),
			CounterAccount:     secondBuyerAccount.Address(),
			CounterAssetType:   secondBoughtAssetType,
			CounterAssetCode:   secondBoughtAssetCode,
			CounterAssetIssuer: secondBoughtAssetIssuer,
			CounterAmount:      second.Trade.AmountBought,
			BaseIsSeller:       true,
			PriceN:             null.NewInt(int64(second.SellPrice.N), true),
			PriceD:             null.NewInt(int64(second.SellPrice.D), true),
		},
		Trade{
			HistoryOperationID: third.HistoryOperationID,
			Order:              third.Order,
			LedgerCloseTime:    third.LedgerCloseTime,
			OfferID:            int64(third.Trade.OfferId),
			BaseOfferID:        newInt64(third.BuyOfferID),
			BaseAccount:        thirdBuyerAccount.Address(),
			BaseAssetType:      thirdBoughtAssetType,
			BaseAssetCode:      thirdBoughtAssetCode,
			BaseAssetIssuer:    thirdBoughtAssetIssuer,
			BaseAmount:         third.Trade.AmountBought,
			CounterOfferID:     newInt64(EncodeOfferId(uint64(third.Trade.OfferId), CoreOfferIDType)),
			CounterAccount:     thirdSellerAccount.Address(),
			CounterAssetType:   thirdSoldAssetType,
			CounterAssetCode:   thirdSoldAssetCode,
			CounterAssetIssuer: thirdSoldAssetIssuer,
			CounterAmount:      third.Trade.AmountSold,
			BaseIsSeller:       false,
			PriceN:             null.NewInt(int64(third.SellPrice.D), true),
			PriceD:             null.NewInt(int64(third.SellPrice.N), true),
		},
	}
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
