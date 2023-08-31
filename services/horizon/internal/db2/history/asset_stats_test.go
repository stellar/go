package history

import (
	"database/sql"
	"sort"
	"testing"

	"golang.org/x/exp/slices"

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
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "0",
				AuthorizedToMaintainLiabilities: "0",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
				Unauthorized:                    "0",
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
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "23",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
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
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
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

func TestAssetContractStats(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	c1 := ContractStatRow{
		ContractID: []byte{1},
		Stat: ContractStat{
			Balance: "100",
			Holders: 2,
		},
	}
	c2 := ContractStatRow{
		ContractID: []byte{2},
		Stat: ContractStat{
			Balance: "40",
			Holders: 1,
		},
	}
	c3 := ContractStatRow{
		ContractID: []byte{3},
		Stat: ContractStat{
			Balance: "900",
			Holders: 12,
		},
	}

	rows := []ContractStatRow{c1, c2, c3}
	tt.Assert.NoError(q.InsertAssetContractStats(tt.Ctx, rows, 5))

	for _, row := range rows {
		result, err := q.GetAssetContractStat(tt.Ctx, row.ContractID)
		tt.Assert.NoError(err)
		tt.Assert.Equal(result, row)
	}

	c2.Stat.Holders = 3
	c2.Stat.Balance = "20"
	numRows, err := q.UpdateAssetContractStat(tt.Ctx, c2)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), numRows)
	row, err := q.GetAssetContractStat(tt.Ctx, c2.ContractID)
	tt.Assert.NoError(err)
	tt.Assert.Equal(c2, row)

	numRows, err = q.RemoveAssetContractStat(tt.Ctx, c3.ContractID)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), numRows)

	_, err = q.GetAssetContractStat(tt.Ctx, c3.ContractID)
	tt.Assert.Equal(sql.ErrNoRows, err)

	for _, row := range []ContractStatRow{c1, c2} {
		result, err := q.GetAssetContractStat(tt.Ctx, row.ContractID)
		tt.Assert.NoError(err)
		tt.Assert.Equal(result, row)
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
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
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
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "23",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
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
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
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
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "23",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "4",
				LiquidityPools:                  "5",
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
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
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
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
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
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
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
	assetStat.Amount = "23"
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
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
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
		},
		Balances: ExpAssetStatBalances{
			Authorized:                      "1",
			AuthorizedToMaintainLiabilities: "2",
			Unauthorized:                    "3",
			ClaimableBalances:               "4",
			LiquidityPools:                  "5",
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

func TestGetAssetStatsFiltersAndCursor(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &Q{tt.HorizonSession()}
	zero := ContractStat{
		Balance: "0",
		Holders: 0,
	}
	usdAssetStat := AssetAndContractStat{
		ExpAssetStat: ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "USD",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      2,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
			},
			Amount:      "1",
			NumAccounts: 2,
		},
		Contracts: zero,
	}
	etherAssetStat := AssetAndContractStat{
		ExpAssetStat: ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "ETHER",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      1,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "23",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
			},
			Amount:      "23",
			NumAccounts: 1,
		},
		Contracts: zero,
	}
	otherUSDAssetStat := AssetAndContractStat{
		ExpAssetStat: ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			AssetCode:   "USD",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      2,
				AuthorizedToMaintainLiabilities: 3,
				Unauthorized:                    4,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "1",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
			},
			Amount:      "1",
			NumAccounts: 2,
		},
		Contracts: zero,
	}
	eurAssetStat := AssetAndContractStat{
		ExpAssetStat: ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			AssetCode:   "EUR",
			Accounts: ExpAssetStatAccounts{
				Authorized:                      3,
				AuthorizedToMaintainLiabilities: 2,
				Unauthorized:                    4,
			},
			Balances: ExpAssetStatBalances{
				Authorized:                      "111",
				AuthorizedToMaintainLiabilities: "2",
				Unauthorized:                    "3",
				ClaimableBalances:               "1",
				LiquidityPools:                  "2",
			},
			Amount:      "111",
			NumAccounts: 3,
		},
		Contracts: ContractStat{
			Balance: "120",
			Holders: 3,
		},
	}
	eurAssetStat.SetContractID([32]byte{})
	assetStats := []AssetAndContractStat{
		etherAssetStat,
		eurAssetStat,
		otherUSDAssetStat,
		usdAssetStat,
	}
	for _, assetStat := range assetStats {
		numChanged, err := q.InsertAssetStat(tt.Ctx, assetStat.ExpAssetStat)
		tt.Assert.NoError(err)
		tt.Assert.Equal(numChanged, int64(1))
		if assetStat.Contracts != zero {
			numChanged, err = q.InsertAssetContractStat(tt.Ctx, ContractStatRow{
				ContractID: *assetStat.ContractID,
				Stat:       assetStat.Contracts,
			})
			tt.Assert.NoError(err)
			tt.Assert.Equal(numChanged, int64(1))
		}
	}

	// insert contract stat which has no corresponding asset stat row
	// to test that it isn't included in the results
	numChanged, err := q.InsertAssetContractStat(tt.Ctx, ContractStatRow{
		ContractID: []byte{1},
		Stat: ContractStat{
			Balance: "400",
			Holders: 30,
		},
	})
	tt.Assert.NoError(err)
	tt.Assert.Equal(numChanged, int64(1))

	for _, testCase := range []struct {
		name        string
		assetCode   string
		assetIssuer string
		cursor      string
		order       string
		expected    []AssetAndContractStat
	}{
		{
			"no filter without cursor",
			"",
			"",
			"",
			"asc",
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
				etherAssetStat,
			},
		},
		{
			"no filter with cursor and offset descending including eur",
			"",
			"",
			"EUR_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum4",
			"desc",
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
				usdAssetStat,
			},
		},
		{
			"filter on code with cursor descending",
			"USD",
			"",
			"USD_GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H_credit_alphanum4",
			"desc",
			[]AssetAndContractStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on issuer without cursor",
			"",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"",
			"asc",
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on issuer with cursor descending",
			"",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"USD_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"desc",
			[]AssetAndContractStat{
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
			[]AssetAndContractStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on both code and issuer with cursor",
			"USD",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"USC_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"asc",
			[]AssetAndContractStat{
				otherUSDAssetStat,
			},
		},
		{
			"filter on both code and issuer with cursor descending",
			"USD",
			"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
			"USE_GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2_credit_alphanum4",
			"desc",
			[]AssetAndContractStat{
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
				slices.Reverse(results)
				tt.Assert.Equal(testCase.expected, results)
			}
		})
	}
}
