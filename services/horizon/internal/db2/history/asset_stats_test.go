package history

import (
	"database/sql"
	"sort"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestAssetStatContracts(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	assetStats := []ExpAssetStat{
		{
			AssetType: xdr.AssetTypeAssetTypeNative,
			Accounts: ExpAssetStatAccounts{
				Authorized:                      0,
				AuthorizedToMaintainLiabilities: 0,
				ClaimableBalances:               0,
				LiquidityPools:                  0,
				Unauthorized:                    0,
				Contracts:                       0,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "0",
				AuthorizedToMaintainLiabilities: "0",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
				Unauthorized:                    "0",
				Contracts:                       "0",
			},
			Amount:      "0",
			NumAccounts: 0,
		},
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "ETHER",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      1,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
				Contracts:                       7,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "23",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
				Contracts:                       "60",
			},
			Amount:      "23",
			NumAccounts: 1,
		},
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "USD",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      2,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
				Contracts:                       8,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
				Contracts:                       "90",
			},
			Amount:      "1",
			NumAccounts: 2,
		},
	}
	var contractID [32]byte
	for i := 0; i < 2; i++ {
		assetStats[i].SetContractID(contractID)
		contractID[0]++
	}
	tt.Assert.NoError(q.InsertAssetStats(tt.Ctx, assetStats, 1))

	contractID[0] = 0
	for i := 0; i < 2; i++ {
		var assetStat ExpAssetStat
		assetStat, err := q.GetAssetStatByContract(tt.Ctx, contractID)
		tt.Assert.NoError(err)
		tt.Assert.True(assetStat.Equals(assetStats[i]))
		contractID[0]++
	}

	contractIDs := make([][32]byte, 2)
	contractIDs[1][0]++
	rows, err := q.GetAssetStatByContracts(tt.Ctx, contractIDs)
	tt.Assert.NoError(err)
	tt.Assert.Len(rows, 2)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].AssetCode < rows[j].AssetCode
	})

	for i, row := range rows {
		tt.Assert.True(row.Equals(assetStats[i]))
	}

	usd := assetStats[2]
	usd.SetContractID([32]byte{})
	_, err = q.UpdateAssetStat(tt.Ctx, usd)
	tt.Assert.EqualError(err, "exec failed: pq: duplicate key value violates unique constraint \"exp_asset_stats_contract_id_key\"")

	usd.SetContractID([32]byte{2})
	rowsUpdated, err := q.UpdateAssetStat(tt.Ctx, usd)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsUpdated)

	assetStats[2] = usd
	contractID = [32]byte{}
	for i := 0; i < 3; i++ {
		var assetStat ExpAssetStat
		assetStat, err = q.GetAssetStatByContract(tt.Ctx, contractID)
		tt.Assert.NoError(err)
		tt.Assert.True(assetStat.Equals(assetStats[i]))
		contractID[0]++
	}

	contractIDs = [][32]byte{{}, {1}, {2}}
	rows, err = q.GetAssetStatByContracts(tt.Ctx, contractIDs)
	tt.Assert.NoError(err)
	tt.Assert.Len(rows, 3)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].AssetCode < rows[j].AssetCode
	})

	for i, row := range rows {
		tt.Assert.True(row.Equals(assetStats[i]))
	}
}

func TestInsertAssetStats(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	tt.Assert.NoError(q.InsertAssetStats(tt.Ctx, []ExpAssetStat{}, 1))

	assetStats := []ExpAssetStat{
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "USD",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      2,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
				Contracts:                       0,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
				Contracts:                       "0",
			},
			Amount:      "1",
			NumAccounts: 2,
		},
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "ETHER",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      1,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
				Contracts:                       0,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "23",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
				Contracts:                       "0",
			},
			Amount:      "23",
			NumAccounts: 1,
		},
	}
	tt.Assert.NoError(q.InsertAssetStats(tt.Ctx, assetStats, 1))

	for _, assetStat := range assetStats {
		got, err := q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
		tt.Assert.NoError(err)
		tt.Assert.Equal(got, assetStat)
	}
}

func TestInsertAssetStat(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	assetStats := []ExpAssetStat{
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "USD",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      2,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
				Contracts:                       0,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
				Contracts:                       "0",
			},
			Amount:      "1",
			NumAccounts: 2,
		},
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "ETHER",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      1,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
				Contracts:                       0,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "23",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
				Contracts:                       "0",
			},
			Amount:      "23",
			NumAccounts: 1,
		},
	}

	for _, assetStat := range assetStats {
		numChanged, err := q.InsertAssetStat(tt.Ctx, assetStat)
		tt.Assert.NoError(err)
		tt.Assert.Equal(numChanged, int64(1))

		got, err := q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
		tt.Assert.NoError(err)
		tt.Assert.Equal(got, assetStat)
	}
}

func TestInsertAssetStatAlreadyExistsError(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	assetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "USD",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      2,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 2,
	}

	numChanged, err := q.InsertAssetStat(tt.Ctx, assetStat)
	tt.Assert.NoError(err)
	tt.Assert.Equal(numChanged, int64(1))

	numChanged, err = q.InsertAssetStat(tt.Ctx, assetStat)
	tt.Assert.Error(err)
	tt.Assert.Equal(numChanged, int64(0))

	assetStat.NumAccounts = 4
	assetStat.Amount = "3"
	numChanged, err = q.InsertAssetStat(tt.Ctx, assetStat)
	tt.Assert.Error(err)
	tt.Assert.Equal(numChanged, int64(0))

	assetStat.NumAccounts = 2
	assetStat.Amount = "1"
	got, err := q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.NoError(err)
	tt.Assert.Equal(got, assetStat)
}

func TestUpdateAssetStatDoesNotExistsError(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	assetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "USD",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      2,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 2,
	}

	numChanged, err := q.UpdateAssetStat(tt.Ctx, assetStat)
	tt.Assert.Nil(err)
	tt.Assert.Equal(numChanged, int64(0))

	_, err = q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.Equal(err, sql.ErrNoRows)
}

func TestUpdateStat(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &Q{tt.HorizonSession()}

	assetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "USD",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      2,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 2,
	}

	numChanged, err := q.InsertAssetStat(tt.Ctx, assetStat)
	tt.Assert.NoError(err)
	tt.Assert.Equal(numChanged, int64(1))

	got, err := q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.NoError(err)
	tt.Assert.Equal(got, assetStat)

	assetStat.NumAccounts = 50
	assetStat.Accounts.Contracts = 4
	assetStat.Amount = "23"
	assetStat.Balances.Contracts = "56"
	assetStat.SetContractID([32]byte{23})

	numChanged, err = q.UpdateAssetStat(tt.Ctx, assetStat)
	tt.Assert.Nil(err)
	tt.Assert.Equal(numChanged, int64(1))

	got, err = q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.NoError(err)
	tt.Assert.True(got.Equals(assetStat))
}

func TestGetAssetStatDoesNotExist(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	assetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "USD",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      2,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 2,
	}

	_, err := q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.Equal(err, sql.ErrNoRows)
}

func TestRemoveAssetStat(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &Q{tt.HorizonSession()}

	assetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "USD",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      2,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 2,
	}

	numChanged, err := q.RemoveAssetStat(tt.Ctx,
		assetStat.AssetType,
		assetStat.AssetCode,
		assetStat.AssetIssuer,
	)
	tt.Assert.Nil(err)
	tt.Assert.Equal(numChanged, int64(0))

	numChanged, err = q.InsertAssetStat(tt.Ctx, assetStat)
	tt.Assert.NoError(err)
	tt.Assert.Equal(numChanged, int64(1))

	got, err := q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.NoError(err)
	tt.Assert.Equal(got, assetStat)

	numChanged, err = q.RemoveAssetStat(tt.Ctx,
		assetStat.AssetType,
		assetStat.AssetCode,
		assetStat.AssetIssuer,
	)
	tt.Assert.Nil(err)
	tt.Assert.Equal(numChanged, int64(1))

	_, err = q.GetAssetStat(tt.Ctx, assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.Equal(err, sql.ErrNoRows)
}

func TestGetAssetStatsCursorValidation(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &Q{tt.HorizonSession()}

	for _, testCase := range []struct {
		name          string
		cursor        string
		expectedError string
	}{
		{
			"cursor does not use underscore as serpator",
			"usdc-GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			"invalid asset stats cursor",
		},
		{
			"cursor has no underscore",
			"usdc",
			"invalid asset stats cursor",
		},
		{
			"cursor has too many underscores",
			"usdc_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum4_",
			"invalid asset type in asset stats cursor",
		},
		{
			"issuer in cursor is invalid",
			"usd_abcdefghijklmnopqrstuv_credit_alphanum4",
			"invalid issuer in asset stats cursor",
		},
		{
			"asset type in cursor is invalid",
			"usd_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum",
			"invalid asset type in asset stats cursor",
		},
		{
			"asset code in cursor is too long",
			"abcdefghijklmnopqrstuv_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum12",
			"invalid asset stats cursor",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			page := db2.PageQuery{
				Cursor: testCase.cursor,
				Order:  "asc",
				Limit:  5,
			}
			results, err := q.GetAssetStats(tt.Ctx, "", "", page)
			tt.Assert.Empty(results)
			tt.Assert.NotNil(err)
			tt.Assert.Contains(err.Error(), testCase.expectedError)
		})
	}
}

func TestGetAssetStatsOrderValidation(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &Q{tt.HorizonSession()}

	page := db2.PageQuery{
		Order: "invalid",
		Limit: 5,
	}
	results, err := q.GetAssetStats(tt.Ctx, "", "", page)
	tt.Assert.Empty(results)
	tt.Assert.NotNil(err)
	tt.Assert.Contains(err.Error(), "invalid page order")
}

func reverseAssetStats(a []ExpAssetStat) {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}

func TestGetAssetStatsFiltersAndCursor(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &Q{tt.HorizonSession()}

	usdAssetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "USD",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      2,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 2,
	}
	etherAssetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "ETHER",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      1,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "23",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "23",
		NumAccounts: 1,
	}
	otherUSDAssetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
		AssetCode:   "USD",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      2,
			AuthorizedToMaintainLiabilities: 3,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "1",
		NumAccounts: 2,
	}
	eurAssetStat := ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
		AssetCode:   "EUR",
		Accounts: ExpAssetStatAccounts{
			Authorized:                      3,
			AuthorizedToMaintainLiabilities: 2,
			Unauthorized:                    4,
			Contracts:                       0,
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "111",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "1",
			LiquidityPools:                  "2",
			Contracts:                       "0",
		},
		Amount:      "111",
		NumAccounts: 3,
	}
	assetStats := []ExpAssetStat{
		etherAssetStat,
		eurAssetStat,
		otherUSDAssetStat,
		usdAssetStat,
	}
	for _, assetStat := range assetStats {
		numChanged, err := q.InsertAssetStat(tt.Ctx, assetStat)
		tt.Assert.NoError(err)
		tt.Assert.Equal(numChanged, int64(1))
	}

	for _, testCase := range []struct {
		name        string
		assetCode   string
		assetIssuer string
		cursor      string
		order       string
		expected    []ExpAssetStat
	}{
		{
			"no filter without cursor",
			"",
			"",
			"",
			"asc",
			[]ExpAssetStat{
				etherAssetStat,
				eurAssetStat,
				otherUSDAssetStat,
				usdAssetStat,
			},
		},
		{
			"no filter with cursor",
			"",
			"",
			"ABC_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			[]ExpAssetStat{
				etherAssetStat,
				eurAssetStat,
				otherUSDAssetStat,
				usdAssetStat,
			},
		},
		{
			"no filter with cursor descending",
			"",
			"",
			"ZZZ_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum4",
			"desc",
			[]ExpAssetStat{
				usdAssetStat,
				otherUSDAssetStat,
				eurAssetStat,
				etherAssetStat,
			},
		},
		{
			"no filter with cursor and offset",
			"",
			"",
			"ETHER_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum12",
			"asc",
			[]ExpAssetStat{
				eurAssetStat,
				otherUSDAssetStat,
				usdAssetStat,
			},
		},
		{
			"no filter with cursor and offset descending",
			"",
			"",
			"EUR_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"desc",
			[]ExpAssetStat{
				etherAssetStat,
			},
		},
		{
			"no filter with cursor and offset descending including eur",
			"",
			"",
			"EUR_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum4",
			"desc",
			[]ExpAssetStat{
				eurAssetStat,
				etherAssetStat,
			},
		},
		{
			"filter on code without cursor",
			"USD",
			"",
			"",
			"asc",
			[]ExpAssetStat{
				otherUSDAssetStat,
				usdAssetStat,
			},
		},
		{
			"filter on code with cursor",
			"USD",
			"",
			"USD_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			[]ExpAssetStat{
				usdAssetStat,
			},
		},
		{
			"filter on code with cursor descending",
			"USD",
			"",
			"USD_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum4",
			"desc",
			[]ExpAssetStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on issuer without cursor",
			"",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"",
			"asc",
			[]ExpAssetStat{
				eurAssetStat,
				otherUSDAssetStat,
			},
		},
		{
			"filter on issuer with cursor",
			"",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"EUR_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			[]ExpAssetStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on issuer with cursor descending",
			"",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"USD_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"desc",
			[]ExpAssetStat{
				eurAssetStat,
			},
		},
		{
			"filter on non existent code without cursor",
			"BTC",
			"",
			"",
			"asc",
			nil,
		},
		{
			"filter on non existent code with cursor",
			"BTC",
			"",
			"BTC_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			nil,
		},
		{
			"filter on non existent issuer without cursor",
			"",
			"GAEIHD6U4WSBHJGA2HPWOQ3OQEFQ3Y7QZE2DR76YKZNKPW5YDLYW4UGF",
			"",
			"asc",
			nil,
		},
		{
			"filter on non existent issuer with cursor",
			"",
			"GAEIHD6U4WSBHJGA2HPWOQ3OQEFQ3Y7QZE2DR76YKZNKPW5YDLYW4UGF",
			"AAA_GAEIHD6U4WSBHJGA2HPWOQ3OQEFQ3Y7QZE2DR76YKZNKPW5YDLYW4UGF_credit_alphanum4",
			"asc",
			nil,
		},
		{
			"filter on non existent code and non existent issuer without cursor",
			"BTC",
			"GAEIHD6U4WSBHJGA2HPWOQ3OQEFQ3Y7QZE2DR76YKZNKPW5YDLYW4UGF",
			"",
			"asc",
			nil,
		},
		{
			"filter on non existent code and non existent issuer with cursor",
			"BTC",
			"GAEIHD6U4WSBHJGA2HPWOQ3OQEFQ3Y7QZE2DR76YKZNKPW5YDLYW4UGF",
			"AAA_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			nil,
		},
		{
			"filter on both code and issuer without cursor",
			"USD",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"",
			"asc",
			[]ExpAssetStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on both code and issuer with cursor",
			"USD",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"USC_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			[]ExpAssetStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on both code and issuer with cursor descending",
			"USD",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"USE_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"desc",
			[]ExpAssetStat{
				otherUSDAssetStat,
			},
		},
		{
			"cursor negates filter",
			"USD",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"USD_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			page := db2.PageQuery{
				Order:  testCase.order,
				Cursor: testCase.cursor,
				Limit:  5,
			}
			results, err := q.GetAssetStats(tt.Ctx, testCase.assetCode, testCase.assetIssuer, page)
			tt.Assert.NoError(err)
			tt.Assert.Equal(testCase.expected, results)

			page.Limit = 1
			results, err = q.GetAssetStats(tt.Ctx, testCase.assetCode, testCase.assetIssuer, page)
			tt.Assert.NoError(err)
			if len(testCase.expected) == 0 {
				tt.Assert.Equal(testCase.expected, results)
			} else {
				tt.Assert.Equal(testCase.expected[:1], results)
			}

			if page.Cursor == "" {
				page = page.Invert()
				page.Limit = 5

				results, err = q.GetAssetStats(tt.Ctx, testCase.assetCode, testCase.assetIssuer, page)
				tt.Assert.NoError(err)
				reverseAssetStats(results)
				tt.Assert.Equal(testCase.expected, results)
			}
		})
	}
}
