package history

import (
	"database/sql"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestInsertAssetStats(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	tt.Assert.NoError(q.InsertAssetStats([]ExpAssetStat{}, 1))

	assetStats := []ExpAssetStat{
		ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "USD",
			Amount:      "1",
			NumAccounts: 2,
		},
		ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "ETHER",
			Amount:      "23",
			NumAccounts: 1,
		},
	}
	tt.Assert.NoError(q.InsertAssetStats(assetStats, 1))

	for _, assetStat := range assetStats {
		got, err := q.GetAssetStat(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
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
		ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "USD",
			Amount:      "1",
			NumAccounts: 2,
		},
		ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum12,
			AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			AssetCode:   "ETHER",
			Amount:      "23",
			NumAccounts: 1,
		},
	}

	for _, assetStat := range assetStats {
		numChanged, err := q.InsertAssetStat(assetStat)
		tt.Assert.NoError(err)
		tt.Assert.Equal(numChanged, int64(1))

		got, err := q.GetAssetStat(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
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
		Amount:      "1",
		NumAccounts: 2,
	}

	numChanged, err := q.InsertAssetStat(assetStat)
	tt.Assert.NoError(err)
	tt.Assert.Equal(numChanged, int64(1))

	numChanged, err = q.InsertAssetStat(assetStat)
	tt.Assert.Error(err)
	tt.Assert.Equal(numChanged, int64(0))

	assetStat.NumAccounts = 4
	assetStat.Amount = "3"
	numChanged, err = q.InsertAssetStat(assetStat)
	tt.Assert.Error(err)
	tt.Assert.Equal(numChanged, int64(0))

	assetStat.NumAccounts = 2
	assetStat.Amount = "1"
	got, err := q.GetAssetStat(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
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
		Amount:      "1",
		NumAccounts: 2,
	}

	numChanged, err := q.UpdateAssetStat(assetStat)
	tt.Assert.Nil(err)
	tt.Assert.Equal(numChanged, int64(0))

	_, err = q.GetAssetStat(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
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
		Amount:      "1",
		NumAccounts: 2,
	}

	numChanged, err := q.InsertAssetStat(assetStat)
	tt.Assert.NoError(err)
	tt.Assert.Equal(numChanged, int64(1))

	got, err := q.GetAssetStat(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.NoError(err)
	tt.Assert.Equal(got, assetStat)

	assetStat.NumAccounts = 50
	assetStat.Amount = "23"

	numChanged, err = q.UpdateAssetStat(assetStat)
	tt.Assert.Nil(err)
	tt.Assert.Equal(numChanged, int64(1))

	got, err = q.GetAssetStat(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.NoError(err)
	tt.Assert.Equal(got, assetStat)
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
		Amount:      "1",
		NumAccounts: 2,
	}

	_, err := q.GetAssetStat(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
	tt.Assert.Equal(err, sql.ErrNoRows)
}
