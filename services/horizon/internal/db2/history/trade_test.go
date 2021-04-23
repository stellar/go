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
	tt := test.Start(t)
	tt.Scenario("kahuna")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}
	var trades []Trade

	// All trades
	err := q.Trades().Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx, &trades)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(trades, 4)
	}

	// Paging
	pq := db2.MustPageQuery(trades[0].PagingToken(), false, "asc", 1)
	var pt []Trade

	err = q.Trades().Page(tt.Ctx, pq).Select(tt.Ctx, &pt)
	if tt.Assert.NoError(err) {
		if tt.Assert.Len(pt, 1) {
			tt.Assert.Equal(trades[1], pt[0])
		}
	}

	// Cursor bounds checking
	pq = db2.MustPageQuery("", false, "desc", 1)
	err = q.Trades().Page(tt.Ctx, pq).Select(tt.Ctx, &pt)
	tt.Require.NoError(err)

	// test for asset pairs
	lumen, err := q.GetAssetID(tt.Ctx, xdr.MustNewNativeAsset())
	tt.Require.NoError(err)
	assetUSD, err := q.GetAssetID(tt.Ctx, xdr.MustNewCreditAsset("USD", "GB2QIYT2IAUFMRXKLSLLPRECC6OCOGJMADSPTRK7TGNT2SFR2YGWDARD"))
	tt.Require.NoError(err)
	assetEUR, err := q.GetAssetID(tt.Ctx, xdr.MustNewCreditAsset("EUR", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU"))
	tt.Require.NoError(err)

	err = q.TradesForAssetPair(assetUSD, assetEUR).Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx, &trades)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 0)

	assetUSD, err = q.GetAssetID(tt.Ctx, xdr.MustNewCreditAsset("USD", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU"))
	tt.Require.NoError(err)

	err = q.TradesForAssetPair(lumen, assetUSD).Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx, &trades)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 1)

	tt.Assert.Equal(xdr.Int64(2000000000), trades[0].BaseAmount)
	tt.Assert.Equal(xdr.Int64(1000000000), trades[0].CounterAmount)
	tt.Assert.Equal(true, trades[0].BaseIsSeller)

	// reverse assets
	err = q.TradesForAssetPair(assetUSD, lumen).Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx, &trades)
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
	addressToAccounts, err := q.CreateAccounts(tt.Ctx, accounts, 2)
	tt.Assert.NoError(err)

	accountIDs := []int64{}
	for _, account := range accounts {
		accountIDs = append(accountIDs, addressToAccounts[account])
	}

	assetMap, err := q.CreateAssets(tt.Ctx, assets, 2)
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
		builder.Add(tt.Ctx, first, second, third),
	)
	tt.Assert.NoError(builder.Exec(tt.Ctx))

	var rows []Trade
	tt.Assert.NoError(q.Trades().Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx, &rows))

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

func TestTradesQueryForAccount(t *testing.T) {
	tt := test.Start(t)
	tt.Scenario("kahuna")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}
	tradesQ := q.Trades()
	var trades []Trade

	account := "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU"
	tradesQ.ForAccount(tt.Ctx, account)
	tt.Assert.Equal(int64(15), tradesQ.forAccountID)
	tt.Assert.Equal(int64(0), tradesQ.forOfferID)

	tradesQ.Page(tt.Ctx, db2.MustPageQuery("", false, "desc", 100))
	_, _, err := tradesQ.sql.ToSql()
	// q.sql was reset in Page so should return error
	tt.Assert.EqualError(err, "select statements must have at least one result column")

	expectedRawSQL := `(SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.offer_id, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.base_account_id = ? AND (
				htrd.history_operation_id <= ?
			AND (
				htrd.history_operation_id < ? OR
				(htrd.history_operation_id = ? AND htrd.order < ?)
			)) ORDER BY htrd.history_operation_id desc, htrd.order desc) UNION (SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.offer_id, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.counter_account_id = ? AND (
				htrd.history_operation_id <= ?
			AND (
				htrd.history_operation_id < ? OR
				(htrd.history_operation_id = ? AND htrd.order < ?)
			)) ORDER BY htrd.history_operation_id desc, htrd.order desc) ORDER BY history_operation_id desc, "order" desc LIMIT 100`
	tt.Assert.Equal(expectedRawSQL, tradesQ.rawSQL)

	err = tradesQ.Select(tt.Ctx, &trades)
	tt.Assert.NoError(err)
	tt.Assert.Len(trades, 3)

	// Ensure "desc" order and account present
	tt.Assert.Equal(int64(85899350017), trades[0].HistoryOperationID)
	tt.Assert.Equal(account, trades[0].BaseAccount)

	tt.Assert.Equal(int64(81604382721), trades[1].HistoryOperationID)
	tt.Assert.Equal(int32(1), trades[1].Order)
	tt.Assert.Equal(account, trades[1].BaseAccount)

	tt.Assert.Equal(int64(81604382721), trades[2].HistoryOperationID)
	tt.Assert.Equal(int32(0), trades[2].Order)
	tt.Assert.Equal(account, trades[2].CounterAccount)
}

func TestTradesQueryForOffer(t *testing.T) {
	tt := test.Start(t)
	tt.Scenario("kahuna")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}
	tradesQ := q.Trades()
	var trades []Trade

	offerID := int64(2)
	tradesQ.ForOffer(offerID)
	tt.Assert.Equal(int64(0), tradesQ.forAccountID)
	tt.Assert.Equal(int64(2), tradesQ.forOfferID)

	tradesQ.Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100))
	_, _, err := tradesQ.sql.ToSql()
	// q.sql was reset in Page so should return error
	tt.Assert.EqualError(err, "select statements must have at least one result column")

	expectedRawSQL := `(SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.offer_id, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.base_offer_id = ? AND (
				htrd.history_operation_id >= ?
			AND (
				htrd.history_operation_id > ? OR
				(htrd.history_operation_id = ? AND htrd.order > ?)
			)) ORDER BY htrd.history_operation_id asc, htrd.order asc) UNION (SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.offer_id, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.counter_offer_id = ? AND (
				htrd.history_operation_id >= ?
			AND (
				htrd.history_operation_id > ? OR
				(htrd.history_operation_id = ? AND htrd.order > ?)
			)) ORDER BY htrd.history_operation_id asc, htrd.order asc) ORDER BY history_operation_id asc, "order" asc LIMIT 100`
	tt.Assert.Equal(expectedRawSQL, tradesQ.rawSQL)

	err = tradesQ.Select(tt.Ctx, &trades)
	tt.Assert.NoError(err)
	tt.Assert.Len(trades, 2)

	// Ensure "asc" order and offer present
	tt.Assert.Equal(int64(81604382721), trades[0].HistoryOperationID)
	tt.Assert.Equal(offerID, trades[0].OfferID)

	tt.Assert.Equal(int64(85899350017), trades[1].HistoryOperationID)
	tt.Assert.Equal(offerID, trades[1].OfferID)
}
