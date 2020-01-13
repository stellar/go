package history

import (
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
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

func createExpAccountsAndAssets(
	tt *test.T, q *Q, accounts []string, assets []xdr.Asset,
) ([]int64, []int64) {
	addressToAccounts, err := q.CreateExpAccounts(accounts)
	tt.Assert.NoError(err)

	accountIDs := []int64{}
	for _, account := range accounts {
		accountIDs = append(accountIDs, addressToAccounts[account])
	}

	assetMap, err := q.CreateExpAssets(assets)
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

func TestInsertExpTrade(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	addresses := []string{
		"GB2QIYT2IAUFMRXKLSLLPRECC6OCOGJMADSPTRK7TGNT2SFR2YGWDARD",
		"GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU",
	}
	assets := []xdr.Asset{eurAsset, usdAsset, nativeAsset}
	accountIDs, assetIDs := createExpAccountsAndAssets(
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
	tt.Assert.NoError(q.expTrades().Select(&rows))

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

func createTradeRows(
	tt *test.T, q *Q,
	idToAccount map[int64]xdr.AccountId,
	idToAsset map[int64]xdr.Asset,
	entries ...InsertTrade,
) {
	for _, entry := range entries {
		entry.Trade.SellerId = idToAccount[entry.SellerAccountID]
		entry.Trade.AssetSold = idToAsset[entry.SoldAssetID]
		entry.Trade.AssetBought = idToAsset[entry.BoughtAssetID]

		err := q.InsertTrade(
			entry.HistoryOperationID,
			entry.Order,
			idToAccount[entry.BuyerAccountID],
			entry.BuyOfferExists,
			xdr.OfferEntry{OfferId: xdr.Int64(entry.BuyOfferID)},
			entry.Trade,
			entry.SellPrice,
			supportTime.MillisFromSeconds(entry.LedgerCloseTime.Unix()),
		)
		tt.Assert.NoError(err)
	}
}

func TestCheckExpTrades(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sequence := int32(56)
	valid, err := q.CheckExpTrades(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	addresses := []string{
		"GB2QIYT2IAUFMRXKLSLLPRECC6OCOGJMADSPTRK7TGNT2SFR2YGWDARD",
		"GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU",
	}
	assets := []xdr.Asset{
		xdr.MustNewCreditAsset("CHF", issuer.Address()),
		eurAsset, usdAsset, nativeAsset,
		xdr.MustNewCreditAsset("BTC", issuer.Address()),
	}

	expAccountIDs, expAssetIDs := createExpAccountsAndAssets(
		tt, q,
		addresses,
		assets,
	)

	chfAssetID, btcAssetID := expAssetIDs[0], expAssetIDs[4]
	assets = assets[1:4]
	expAssetIDs = expAssetIDs[1:4]

	idToAccount := buildIDtoAccountMapping(addresses, expAccountIDs)
	idToAsset := buildIDtoAssetMapping(assets, expAssetIDs)

	first, second, third := createInsertTrades(
		expAccountIDs, expAssetIDs, sequence,
	)

	builder := q.NewTradeBatchInsertBuilder(1)
	tt.Assert.NoError(
		builder.Add(first, second, third),
	)
	tt.Assert.NoError(builder.Exec())

	valid, err = q.CheckExpTrades(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	// create different asset id ordering in history_assets compared to exp_history_assets
	_, err = q.GetCreateAssetID(assets[1])
	tt.Assert.NoError(err)
	_, err = q.GetCreateAssetID(assets[0])
	tt.Assert.NoError(err)
	_, err = q.GetCreateAssetID(assets[2])
	tt.Assert.NoError(err)
	createTradeRows(
		tt, q, idToAccount, idToAsset, first, second, third,
	)

	valid, err = q.CheckExpTrades(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	tradeForOtherLedger, _, _ := createInsertTrades(
		expAccountIDs, expAssetIDs, sequence+1,
	)
	tt.Assert.NoError(
		builder.Add(tradeForOtherLedger),
	)
	tt.Assert.NoError(builder.Exec())

	valid, err = q.CheckExpTrades(sequence)
	tt.Assert.NoError(err)
	tt.Assert.True(valid)

	newAddress := "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY"
	newAccounts, err := q.CreateExpAccounts([]string{newAddress})
	tt.Assert.NoError(err)
	newAccountID := newAccounts[newAddress]

	for fieldName, value := range map[string]interface{}{
		"ledger_closed_at":   time.Now().Add(time.Hour),
		"offer_id":           67,
		"base_offer_id":      67,
		"base_account_id":    newAccountID,
		"base_asset_id":      chfAssetID,
		"base_amount":        67,
		"counter_offer_id":   67,
		"counter_account_id": newAccountID,
		"counter_asset_id":   btcAssetID,
		"counter_amount":     67,
		"base_is_seller":     second.SoldAssetID >= second.BoughtAssetID,
		"price_n":            67,
		"price_d":            67,
	} {
		updateSQL := sq.Update("exp_history_trades").
			Set(fieldName, value).
			Where(
				"history_operation_id = ? AND \"order\" = ?",
				second.HistoryOperationID, second.Order,
			)
		_, err = q.Exec(updateSQL)
		tt.Assert.NoError(err)

		valid, err = q.CheckExpTrades(sequence)
		tt.Assert.NoError(err)
		tt.Assert.False(valid)

		_, err = q.Exec(sq.Delete("exp_history_trades").
			Where(
				"history_operation_id = ? AND \"order\" = ?",
				second.HistoryOperationID, second.Order,
			))
		tt.Assert.NoError(err)

		tt.Assert.NoError(
			builder.Add(second),
		)
		tt.Assert.NoError(builder.Exec())

		valid, err := q.CheckExpTrades(sequence)
		tt.Assert.NoError(err)
		tt.Assert.True(valid)
	}
}
