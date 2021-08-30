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
	// TODO fix in https://github.com/stellar/go/issues/3835
	t.Skip()
	tt := test.Start(t)
	tt.Scenario("kahuna")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	// All trades
	trades, err := q.Trades().Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx)
	if tt.Assert.NoError(err) {
		tt.Assert.Len(trades, 4)
	}

	// Paging
	pq := db2.MustPageQuery(trades[0].PagingToken(), false, "asc", 1)

	var pt []Trade
	pt, err = q.Trades().Page(tt.Ctx, pq).Select(tt.Ctx)
	if tt.Assert.NoError(err) {
		if tt.Assert.Len(pt, 1) {
			tt.Assert.Equal(trades[1], pt[0])
		}
	}

	// Cursor bounds checking
	pq = db2.MustPageQuery("", false, "desc", 1)
	pt, err = q.Trades().Page(tt.Ctx, pq).Select(tt.Ctx)
	tt.Require.NoError(err)

	// test for asset pairs
	lumen, err := q.GetAssetID(tt.Ctx, xdr.MustNewNativeAsset())
	tt.Require.NoError(err)
	assetUSD, err := q.GetAssetID(tt.Ctx, xdr.MustNewCreditAsset("USD", "GB2QIYT2IAUFMRXKLSLLPRECC6OCOGJMADSPTRK7TGNT2SFR2YGWDARD"))
	tt.Require.NoError(err)
	assetEUR, err := q.GetAssetID(tt.Ctx, xdr.MustNewCreditAsset("EUR", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU"))
	tt.Require.NoError(err)

	trades, err = q.TradesForAssetPair(assetUSD, assetEUR).Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 0)

	assetUSD, err = q.GetAssetID(tt.Ctx, xdr.MustNewCreditAsset("USD", "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU"))
	tt.Require.NoError(err)

	trades, err = q.TradesForAssetPair(lumen, assetUSD).Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 1)

	tt.Assert.Equal(int64(2000000000), trades[0].BaseAmount)
	tt.Assert.Equal(int64(1000000000), trades[0].CounterAmount)
	tt.Assert.Equal(true, trades[0].BaseIsSeller)

	// reverse assets
	trades, err = q.TradesForAssetPair(assetUSD, lumen).Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx)
	tt.Require.NoError(err)
	tt.Assert.Len(trades, 1)

	tt.Assert.Equal(int64(1000000000), trades[0].BaseAmount)
	tt.Assert.Equal(int64(2000000000), trades[0].CounterAmount)
	tt.Assert.Equal(false, trades[0].BaseIsSeller)
}

func createInsertTrades(
	accountIDs, assetIDs, poolIDs []int64, ledger int32,
) []InsertTrade {
	first := InsertTrade{
		HistoryOperationID: toid.New(ledger, 1, 1).ToInt64(),
		Order:              1,
		LedgerCloseTime:    supportTime.MillisFromSeconds(time.Now().Unix()).ToTime(),
		CounterOfferID:     null.IntFrom(32145),
		BaseAccountID:      null.IntFrom(accountIDs[0]),
		CounterAccountID:   null.IntFrom(accountIDs[1]),
		BaseAssetID:        assetIDs[0],
		CounterAssetID:     assetIDs[1],
		BaseOfferID:        null.IntFrom(214515),
		BaseIsSeller:       true,
		BaseAmount:         7986,
		CounterAmount:      896,
		PriceN:             1,
		PriceD:             3,
	}

	second := first
	second.CounterOfferID = null.Int{}
	second.Order = 2

	third := InsertTrade{
		HistoryOperationID: toid.New(ledger, 2, 1).ToInt64(),
		Order:              1,
		LedgerCloseTime:    time.Now().UTC(),
		CounterOfferID:     null.IntFrom(2),
		BaseAccountID:      null.IntFrom(accountIDs[0]),
		CounterAccountID:   null.IntFrom(accountIDs[1]),
		BaseAssetID:        assetIDs[1],
		CounterAssetID:     assetIDs[2],
		BaseOfferID:        null.IntFrom(7),
		BaseIsSeller:       false,
		BaseAmount:         123,
		CounterAmount:      6,
		PriceN:             1156,
		PriceD:             3,
	}

	fourth := InsertTrade{
		HistoryOperationID:  toid.New(ledger, 2, 2).ToInt64(),
		Order:               3,
		LedgerCloseTime:     time.Now().UTC(),
		CounterAssetID:      assetIDs[4],
		CounterAmount:       675,
		CounterAccountID:    null.IntFrom(accountIDs[0]),
		LiquidityPoolFee:    null.IntFrom(xdr.LiquidityPoolFeeV18),
		BaseAssetID:         assetIDs[3],
		BaseAmount:          981,
		BaseLiquidityPoolID: null.IntFrom(poolIDs[0]),
		BaseIsSeller:        true,
		PriceN:              675,
		PriceD:              981,
	}

	return []InsertTrade{
		first,
		second,
		third,
		fourth,
	}
}

func createHistoryIDs(
	tt *test.T, q *Q, accounts []string, assets []xdr.Asset, pools []string,
) ([]int64, []int64, []int64) {
	addressToAccounts, err := q.CreateAccounts(tt.Ctx, accounts, len(accounts))
	tt.Assert.NoError(err)

	accountIDs := []int64{}
	for _, account := range accounts {
		accountIDs = append(accountIDs, addressToAccounts[account])
	}

	assetMap, err := q.CreateAssets(tt.Ctx, assets, len(assets))
	tt.Assert.NoError(err)

	assetIDs := []int64{}
	for _, asset := range assets {
		assetIDs = append(assetIDs, assetMap[asset.String()].ID)
	}

	poolsMap, err := q.CreateHistoryLiquidityPools(tt.Ctx, pools, len(pools))
	tt.Assert.NoError(err)
	poolIDs := []int64{}
	for _, pool := range pools {
		poolIDs = append(poolIDs, poolsMap[pool])
	}

	return accountIDs, assetIDs, poolIDs
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
	assets := []xdr.Asset{
		eurAsset,
		usdAsset,
		nativeAsset,
		xdr.MustNewCreditAsset("JPY", addresses[0]),
		xdr.MustNewCreditAsset("CHF", addresses[1]),
	}
	pools := []string{"pool1"}
	accountIDs, assetIDs, poolIDs := createHistoryIDs(
		tt, q,
		addresses,
		assets,
		pools,
	)

	inserts := createInsertTrades(accountIDs, assetIDs, poolIDs, 3)

	builder := q.NewTradeBatchInsertBuilder(1)
	tt.Assert.NoError(
		builder.Add(tt.Ctx, inserts...),
	)
	tt.Assert.NoError(builder.Exec(tt.Ctx))

	rows, err := q.Trades().Page(tt.Ctx, db2.MustPageQuery("", false, "asc", 100)).Select(tt.Ctx)
	tt.Assert.NoError(err)

	idToAccount := buildIDtoAccountMapping(addresses, accountIDs)
	idToAsset := buildIDtoAssetMapping(assets, assetIDs)

	firstSellerAccount := idToAccount[inserts[0].BaseAccountID.Int64]
	firstBuyerAccount := idToAccount[inserts[0].CounterAccountID.Int64]
	var firstSoldAssetType, firstSoldAssetCode, firstSoldAssetIssuer string
	idToAsset[inserts[0].BaseAssetID].MustExtract(
		&firstSoldAssetType, &firstSoldAssetCode, &firstSoldAssetIssuer,
	)
	var firstBoughtAssetType, firstBoughtAssetCode, firstBoughtAssetIssuer string
	idToAsset[inserts[0].CounterAssetID].MustExtract(
		&firstBoughtAssetType, &firstBoughtAssetCode, &firstBoughtAssetIssuer,
	)

	secondSellerAccount := idToAccount[inserts[1].BaseAccountID.Int64]
	secondBuyerAccount := idToAccount[inserts[1].CounterAccountID.Int64]
	var secondSoldAssetType, secondSoldAssetCode, secondSoldAssetIssuer string
	idToAsset[inserts[1].BaseAssetID].MustExtract(
		&secondSoldAssetType, &secondSoldAssetCode, &secondSoldAssetIssuer,
	)
	var secondBoughtAssetType, secondBoughtAssetCode, secondBoughtAssetIssuer string
	idToAsset[inserts[1].CounterAssetID].MustExtract(
		&secondBoughtAssetType, &secondBoughtAssetCode, &secondBoughtAssetIssuer,
	)

	thirdSellerAccount := idToAccount[inserts[2].BaseAccountID.Int64]
	thirdBuyerAccount := idToAccount[inserts[2].CounterAccountID.Int64]
	var thirdSoldAssetType, thirdSoldAssetCode, thirdSoldAssetIssuer string
	idToAsset[inserts[2].BaseAssetID].MustExtract(
		&thirdSoldAssetType, &thirdSoldAssetCode, &thirdSoldAssetIssuer,
	)
	var thirdBoughtAssetType, thirdBoughtAssetCode, thirdBoughtAssetIssuer string
	idToAsset[inserts[2].CounterAssetID].MustExtract(
		&thirdBoughtAssetType, &thirdBoughtAssetCode, &thirdBoughtAssetIssuer,
	)

	var fourthSoldAssetType, fourthSoldAssetCode, fourthSoldAssetIssuer string
	idToAsset[inserts[3].BaseAssetID].MustExtract(
		&fourthSoldAssetType, &fourthSoldAssetCode, &fourthSoldAssetIssuer,
	)
	var fourthBoughtAssetType, fourthBoughtAssetCode, fourthBoughtAssetIssuer string
	idToAsset[inserts[3].CounterAssetID].MustExtract(
		&fourthBoughtAssetType, &fourthBoughtAssetCode, &fourthBoughtAssetIssuer,
	)

	expected := []Trade{
		{
			HistoryOperationID: inserts[0].HistoryOperationID,
			Order:              inserts[0].Order,
			LedgerCloseTime:    inserts[0].LedgerCloseTime,
			BaseOfferID:        inserts[0].BaseOfferID,
			BaseAccount:        null.StringFrom(firstSellerAccount.Address()),
			BaseAssetType:      firstSoldAssetType,
			BaseAssetIssuer:    firstSoldAssetIssuer,
			BaseAssetCode:      firstSoldAssetCode,
			BaseAmount:         inserts[0].BaseAmount,
			CounterOfferID:     inserts[0].CounterOfferID,
			CounterAccount:     null.StringFrom(firstBuyerAccount.Address()),
			CounterAssetType:   firstBoughtAssetType,
			CounterAssetIssuer: firstBoughtAssetIssuer,
			CounterAssetCode:   firstBoughtAssetCode,
			CounterAmount:      inserts[0].CounterAmount,
			BaseIsSeller:       true,
			PriceN:             null.IntFrom(inserts[0].PriceN),
			PriceD:             null.IntFrom(inserts[0].PriceD),
		},
		{
			HistoryOperationID: inserts[1].HistoryOperationID,
			Order:              inserts[1].Order,
			LedgerCloseTime:    inserts[1].LedgerCloseTime,
			BaseOfferID:        inserts[1].BaseOfferID,
			BaseAccount:        null.StringFrom(secondSellerAccount.Address()),
			BaseAssetType:      secondSoldAssetType,
			BaseAssetIssuer:    secondSoldAssetIssuer,
			BaseAssetCode:      secondSoldAssetCode,
			BaseAmount:         inserts[1].BaseAmount,
			CounterOfferID:     null.Int{},
			CounterAccount:     null.StringFrom(secondBuyerAccount.Address()),
			CounterAssetType:   secondBoughtAssetType,
			CounterAssetCode:   secondBoughtAssetCode,
			CounterAssetIssuer: secondBoughtAssetIssuer,
			CounterAmount:      inserts[1].CounterAmount,
			BaseIsSeller:       true,
			PriceN:             null.IntFrom(inserts[1].PriceN),
			PriceD:             null.IntFrom(inserts[1].PriceD),
		},
		{
			HistoryOperationID: inserts[2].HistoryOperationID,
			Order:              inserts[2].Order,
			LedgerCloseTime:    inserts[2].LedgerCloseTime,
			BaseOfferID:        inserts[2].BaseOfferID,
			BaseAccount:        null.StringFrom(thirdSellerAccount.Address()),
			BaseAssetType:      thirdSoldAssetType,
			BaseAssetCode:      thirdSoldAssetCode,
			BaseAssetIssuer:    thirdSoldAssetIssuer,
			BaseAmount:         inserts[2].BaseAmount,
			CounterOfferID:     inserts[2].CounterOfferID,
			CounterAccount:     null.StringFrom(thirdBuyerAccount.Address()),
			CounterAssetType:   thirdBoughtAssetType,
			CounterAssetCode:   thirdBoughtAssetCode,
			CounterAssetIssuer: thirdBoughtAssetIssuer,
			CounterAmount:      inserts[2].CounterAmount,
			BaseIsSeller:       false,
			PriceN:             null.IntFrom(inserts[2].PriceN),
			PriceD:             null.IntFrom(inserts[2].PriceD),
		},
		{
			HistoryOperationID:  inserts[3].HistoryOperationID,
			Order:               inserts[3].Order,
			LedgerCloseTime:     inserts[3].LedgerCloseTime,
			BaseOfferID:         inserts[3].BaseOfferID,
			BaseAssetType:       fourthSoldAssetType,
			BaseAssetCode:       fourthSoldAssetCode,
			BaseAssetIssuer:     fourthSoldAssetIssuer,
			BaseLiquidityPoolID: null.StringFrom(pools[0]),
			BaseAmount:          inserts[3].BaseAmount,
			CounterOfferID:      null.Int{},
			CounterAccount:      null.StringFrom(thirdSellerAccount.Address()),
			CounterAssetType:    fourthBoughtAssetType,
			CounterAssetCode:    fourthBoughtAssetCode,
			CounterAssetIssuer:  fourthBoughtAssetIssuer,
			CounterAmount:       inserts[3].CounterAmount,
			BaseIsSeller:        inserts[3].BaseIsSeller,
			LiquidityPoolFee:    inserts[3].LiquidityPoolFee,
			PriceN:              null.IntFrom(inserts[3].PriceN),
			PriceD:              null.IntFrom(inserts[3].PriceD),
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
	// TODO fix in https://github.com/stellar/go/issues/3835
	t.Skip()
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

	expectedRawSQL := `(SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.base_account_id = ? AND (
				htrd.history_operation_id <= ?
			AND (
				htrd.history_operation_id < ? OR
				(htrd.history_operation_id = ? AND htrd.order < ?)
			)) ORDER BY htrd.history_operation_id desc, htrd.order desc) UNION (SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.counter_account_id = ? AND (
				htrd.history_operation_id <= ?
			AND (
				htrd.history_operation_id < ? OR
				(htrd.history_operation_id = ? AND htrd.order < ?)
			)) ORDER BY htrd.history_operation_id desc, htrd.order desc) ORDER BY history_operation_id desc, "order" desc LIMIT 100`
	tt.Assert.Equal(expectedRawSQL, tradesQ.rawSQL)

	trades, err = tradesQ.Select(tt.Ctx)
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
	// TODO fix in https://github.com/stellar/go/issues/3835
	t.Skip()
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

	expectedRawSQL := `(SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.base_offer_id = ? AND (
				htrd.history_operation_id >= ?
			AND (
				htrd.history_operation_id > ? OR
				(htrd.history_operation_id = ? AND htrd.order > ?)
			)) ORDER BY htrd.history_operation_id asc, htrd.order asc) UNION (SELECT history_operation_id, htrd."order", htrd.ledger_closed_at, htrd.base_offer_id, base_accounts.address as base_account, base_assets.asset_type as base_asset_type, base_assets.asset_code as base_asset_code, base_assets.asset_issuer as base_asset_issuer, htrd.base_amount, htrd.counter_offer_id, counter_accounts.address as counter_account, counter_assets.asset_type as counter_asset_type, counter_assets.asset_code as counter_asset_code, counter_assets.asset_issuer as counter_asset_issuer, htrd.counter_amount, htrd.base_is_seller, htrd.price_n, htrd.price_d FROM history_trades htrd JOIN history_accounts base_accounts ON base_account_id = base_accounts.id JOIN history_accounts counter_accounts ON counter_account_id = counter_accounts.id JOIN history_assets base_assets ON base_asset_id = base_assets.id JOIN history_assets counter_assets ON counter_asset_id = counter_assets.id WHERE htrd.counter_offer_id = ? AND (
				htrd.history_operation_id >= ?
			AND (
				htrd.history_operation_id > ? OR
				(htrd.history_operation_id = ? AND htrd.order > ?)
			)) ORDER BY htrd.history_operation_id asc, htrd.order asc) ORDER BY history_operation_id asc, "order" asc LIMIT 100`
	tt.Assert.Equal(expectedRawSQL, tradesQ.rawSQL)

	trades, err = tradesQ.Select(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.Len(trades, 2)

	// Ensure "asc" order and offer present
	tt.Assert.Equal(int64(81604382721), trades[0].HistoryOperationID)
	tt.Assert.Equal(offerID, trades[0].BaseOfferID.Int64)

	tt.Assert.Equal(int64(85899350017), trades[1].HistoryOperationID)
	tt.Assert.Equal(offerID, trades[1].BaseOfferID.Int64)
}
