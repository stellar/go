package history

import (
	"database/sql"
	"testing"

	"github.com/guregu/null"
	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

var (
	trustLineIssuer = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")

	eurTrustLine = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: account1.Data.Account.AccountId,
				Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
				Balance:   30000,
				Limit:     223456789,
				Flags:     1,
				Ext: xdr.TrustLineEntryExt{
					V: 1,
					V1: &xdr.TrustLineEntryV1{
						Liabilities: xdr.Liabilities{
							Buying:  3,
							Selling: 4,
						},
					},
				},
			},
		},
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: &sponsor,
			},
		},
	}

	usdTrustLine = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1235,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: xdr.MustAddress("GCYVFGI3SEQJGBNQQG7YCMFWEYOHK3XPVOVPA6C566PXWN4SN7LILZSM"),
				Asset:     xdr.MustNewCreditAsset("USDUSD", trustLineIssuer.Address()),
				Balance:   10000,
				Limit:     123456789,
				Flags:     0,
				Ext: xdr.TrustLineEntryExt{
					V: 1,
					V1: &xdr.TrustLineEntryV1{
						Liabilities: xdr.Liabilities{
							Buying:  1,
							Selling: 2,
						},
					},
				},
			},
		},
	}

	usdTrustLine2 = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: xdr.MustAddress("GBYSBDAJZMHL5AMD7QXQ3JEP3Q4GLKADWIJURAAHQALNAWD6Z5XF2RAC"),
				Asset:     xdr.MustNewCreditAsset("USDUSD", trustLineIssuer.Address()),
				Balance:   10000,
				Limit:     123456789,
				Flags:     0,
				Ext: xdr.TrustLineEntryExt{
					V: 1,
					V1: &xdr.TrustLineEntryV1{
						Liabilities: xdr.Liabilities{
							Buying:  1,
							Selling: 2,
						},
					},
				},
			},
		},
	}
)

func TestIsAuthorized(t *testing.T) {
	tt := assert.New(t)
	tl := TrustLine{
		Flags: 1,
	}
	tt.True(tl.IsAuthorized())

	tl = TrustLine{
		Flags: 0,
	}
	tt.False(tl.IsAuthorized())
}
func TestInsertTrustLine(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertTrustLine(tt.Ctx, eurTrustLine)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	rows, err = q.InsertTrustLine(tt.Ctx, usdTrustLine)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	keys := []xdr.LedgerKeyTrustLine{
		{Asset: eurTrustLine.Data.TrustLine.Asset, AccountId: eurTrustLine.Data.TrustLine.AccountId},
		{Asset: usdTrustLine.Data.TrustLine.Asset, AccountId: usdTrustLine.Data.TrustLine.AccountId},
	}

	lines, err := q.GetTrustLinesByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 2)

	assert.Equal(t, null.StringFrom(sponsor.Address()), lines[0].Sponsor)
}

func TestUpdateTrustLine(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertTrustLine(tt.Ctx, eurTrustLine)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	modifiedTrustLine := eurTrustLine
	// make sure we don't mutate eurTrustline
	v1Copy := *eurTrustLine.Ext.V1
	v1Copy.SponsoringId = nil
	modifiedTrustLine.Ext.V1 = &v1Copy
	modifiedTrustLine.Data.TrustLine.Balance = 30000

	rows, err = q.UpdateTrustLine(tt.Ctx, modifiedTrustLine)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	keys := []xdr.LedgerKeyTrustLine{
		{Asset: eurTrustLine.Data.TrustLine.Asset, AccountId: eurTrustLine.Data.TrustLine.AccountId},
	}
	lines, err := q.GetTrustLinesByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 1)

	expectedBinary, err := modifiedTrustLine.Data.TrustLine.MarshalBinary()
	assert.NoError(t, err)

	dbEntry := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress(lines[0].AccountID),
		Asset:     xdr.MustNewCreditAsset(lines[0].AssetCode, lines[0].AssetIssuer),
		Balance:   xdr.Int64(lines[0].Balance),
		Limit:     xdr.Int64(lines[0].Limit),
		Flags:     xdr.Uint32(lines[0].Flags),
		Ext: xdr.TrustLineEntryExt{
			V: 1,
			V1: &xdr.TrustLineEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  xdr.Int64(lines[0].BuyingLiabilities),
					Selling: xdr.Int64(lines[0].SellingLiabilities),
				},
			},
		},
	}

	actualBinary, err := dbEntry.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, expectedBinary, actualBinary)
	assert.Equal(t, uint32(1234), lines[0].LastModifiedLedger)
}

func TestUpsertTrustLines(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	// Upserting nothing is no op
	err := q.UpsertTrustLines(tt.Ctx, []xdr.LedgerEntry{})
	assert.NoError(t, err)

	ledgerEntries := []xdr.LedgerEntry{eurTrustLine, usdTrustLine}

	err = q.UpsertTrustLines(tt.Ctx, ledgerEntries)
	assert.NoError(t, err)

	keys := []xdr.LedgerKeyTrustLine{
		{Asset: eurTrustLine.Data.TrustLine.Asset, AccountId: eurTrustLine.Data.TrustLine.AccountId},
	}
	lines, err := q.GetTrustLinesByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 1)

	keys = []xdr.LedgerKeyTrustLine{
		{Asset: usdTrustLine.Data.TrustLine.Asset, AccountId: usdTrustLine.Data.TrustLine.AccountId},
	}
	lines, err = q.GetTrustLinesByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 1)

	modifiedTrustLine := eurTrustLine
	modifiedTrustLine.Data.TrustLine.Balance = 30000

	ledgerEntries = []xdr.LedgerEntry{modifiedTrustLine}

	err = q.UpsertTrustLines(tt.Ctx, ledgerEntries)
	assert.NoError(t, err)

	keys = []xdr.LedgerKeyTrustLine{
		{Asset: eurTrustLine.Data.TrustLine.Asset, AccountId: eurTrustLine.Data.TrustLine.AccountId},
	}
	lines, err = q.GetTrustLinesByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 1)

	expectedBinary, err := modifiedTrustLine.MarshalBinary()
	assert.NoError(t, err)

	dbEntry := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: xdr.MustAddress(lines[0].AccountID),
				Asset:     xdr.MustNewCreditAsset(lines[0].AssetCode, lines[0].AssetIssuer),
				Balance:   xdr.Int64(lines[0].Balance),
				Limit:     xdr.Int64(lines[0].Limit),
				Flags:     xdr.Uint32(lines[0].Flags),
				Ext: xdr.TrustLineEntryExt{
					V: 1,
					V1: &xdr.TrustLineEntryV1{
						Liabilities: xdr.Liabilities{
							Buying:  xdr.Int64(lines[0].BuyingLiabilities),
							Selling: xdr.Int64(lines[0].SellingLiabilities),
						},
					},
				},
			},
		},
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: &sponsor,
			},
		},
	}

	actualBinary, err := dbEntry.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, modifiedTrustLine, dbEntry)
	assert.Equal(t, expectedBinary, actualBinary)
	assert.Equal(t, uint32(1234), lines[0].LastModifiedLedger)

	keys = []xdr.LedgerKeyTrustLine{
		{Asset: usdTrustLine.Data.TrustLine.Asset, AccountId: usdTrustLine.Data.TrustLine.AccountId},
	}
	lines, err = q.GetTrustLinesByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 1)

	expectedBinary, err = usdTrustLine.MarshalBinary()
	assert.NoError(t, err)

	dbEntry = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1235,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: xdr.MustAddress(lines[0].AccountID),
				Asset:     xdr.MustNewCreditAsset(lines[0].AssetCode, lines[0].AssetIssuer),
				Balance:   xdr.Int64(lines[0].Balance),
				Limit:     xdr.Int64(lines[0].Limit),
				Flags:     xdr.Uint32(lines[0].Flags),
				Ext: xdr.TrustLineEntryExt{
					V: 1,
					V1: &xdr.TrustLineEntryV1{
						Liabilities: xdr.Liabilities{
							Buying:  xdr.Int64(lines[0].BuyingLiabilities),
							Selling: xdr.Int64(lines[0].SellingLiabilities),
						},
					},
				},
			},
		},
	}

	actualBinary, err = dbEntry.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, expectedBinary, actualBinary)
	assert.Equal(t, uint32(1235), lines[0].LastModifiedLedger)
}

func TestRemoveTrustLine(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertTrustLine(tt.Ctx, eurTrustLine)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	key := xdr.LedgerKeyTrustLine{Asset: eurTrustLine.Data.TrustLine.Asset, AccountId: eurTrustLine.Data.TrustLine.AccountId}
	rows, err = q.RemoveTrustLine(tt.Ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	lines, err := q.GetTrustLinesByKeys(tt.Ctx, []xdr.LedgerKeyTrustLine{key})
	assert.NoError(t, err)
	assert.Len(t, lines, 0)

	// Doesn't exist anymore
	rows, err = q.RemoveTrustLine(tt.Ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}
func TestGetSortedTrustLinesByAccountsID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.InsertTrustLine(tt.Ctx, eurTrustLine)
	tt.Assert.NoError(err)
	_, err = q.InsertTrustLine(tt.Ctx, usdTrustLine)
	tt.Assert.NoError(err)
	_, err = q.InsertTrustLine(tt.Ctx, usdTrustLine2)
	tt.Assert.NoError(err)

	ids := []string{
		eurTrustLine.Data.TrustLine.AccountId.Address(),
		usdTrustLine.Data.TrustLine.AccountId.Address(),
	}

	records, err := q.GetSortedTrustLinesByAccountIDs(tt.Ctx, ids)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 2)

	m := map[string]xdr.LedgerEntry{
		eurTrustLine.Data.TrustLine.AccountId.Address(): eurTrustLine,
		usdTrustLine.Data.TrustLine.AccountId.Address(): usdTrustLine,
	}

	lastAssetCode := ""
	lastIssuer := records[0].AssetIssuer
	for _, record := range records {
		tt.Assert.LessOrEqual(lastAssetCode, record.AssetCode)
		lastAssetCode = record.AssetCode
		tt.Assert.LessOrEqual(lastIssuer, record.AssetIssuer)
		lastIssuer = record.AssetIssuer
		xtl, ok := m[record.AccountID]
		tt.Assert.True(ok)
		asset := xdr.MustNewCreditAsset(record.AssetCode, record.AssetIssuer)
		tt.Assert.Equal(xtl.Data.TrustLine.Asset, asset)
		tt.Assert.Equal(xtl.Data.TrustLine.AccountId.Address(), record.AccountID)
		delete(m, record.AccountID)
	}

	tt.Assert.Len(m, 0)
}

func TestGetTrustLinesByAccountID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.InsertTrustLine(tt.Ctx, eurTrustLine)
	tt.Assert.NoError(err)

	record, err := q.GetSortedTrustLinesByAccountID(tt.Ctx, eurTrustLine.Data.TrustLine.AccountId.Address())
	tt.Assert.NoError(err)

	asset := xdr.MustNewCreditAsset(record[0].AssetCode, record[0].AssetIssuer)
	tt.Assert.Equal(eurTrustLine.Data.TrustLine.Asset, asset)
	tt.Assert.Equal(eurTrustLine.Data.TrustLine.AccountId.Address(), record[0].AccountID)
	tt.Assert.Equal(int64(eurTrustLine.Data.TrustLine.Balance), record[0].Balance)
	tt.Assert.Equal(int64(eurTrustLine.Data.TrustLine.Limit), record[0].Limit)
	tt.Assert.Equal(uint32(eurTrustLine.Data.TrustLine.Flags), record[0].Flags)
	tt.Assert.Equal(int64(eurTrustLine.Data.TrustLine.Ext.V1.Liabilities.Buying), record[0].BuyingLiabilities)
	tt.Assert.Equal(int64(eurTrustLine.Data.TrustLine.Ext.V1.Liabilities.Selling), record[0].SellingLiabilities)

}

func TestAssetsForAddressRequiresTransaction(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, _, err := q.AssetsForAddress(tt.Ctx, eurTrustLine.Data.TrustLine.AccountId.Address())
	assert.EqualError(t, err, "cannot be called outside of a transaction")

	assert.NoError(t, q.Begin(tt.Ctx))
	defer q.Rollback(tt.Ctx)

	_, _, err = q.AssetsForAddress(tt.Ctx, eurTrustLine.Data.TrustLine.AccountId.Address())
	assert.EqualError(t, err, "should only be called in a repeatable read transaction")
}

func TestAssetsForAddress(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	ledgerEntries := []xdr.LedgerEntry{account1}

	err := q.UpsertAccounts(tt.Ctx, ledgerEntries)
	assert.NoError(t, err)

	_, err = q.InsertTrustLine(tt.Ctx, eurTrustLine)
	tt.Assert.NoError(err)

	brlTrustLine := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: account1.Data.Account.AccountId,
				Asset:     xdr.MustNewCreditAsset("BRL", trustLineIssuer.Address()),
				Balance:   1000,
				Limit:     20000,
				Flags:     1,
				Ext: xdr.TrustLineEntryExt{
					V: 1,
					V1: &xdr.TrustLineEntryV1{
						Liabilities: xdr.Liabilities{
							Buying:  3,
							Selling: 4,
						},
					},
				},
			},
		},
	}

	_, err = q.InsertTrustLine(tt.Ctx, brlTrustLine)
	tt.Assert.NoError(err)

	err = q.BeginTx(tt.Ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	assert.NoError(t, err)
	defer q.Rollback(tt.Ctx)

	assets, balances, err := q.AssetsForAddress(tt.Ctx, usdTrustLine.Data.TrustLine.AccountId.Address())
	tt.Assert.NoError(err)
	tt.Assert.Empty(assets)
	tt.Assert.Empty(balances)

	assets, balances, err = q.AssetsForAddress(tt.Ctx, account1.Data.Account.AccountId.Address())
	tt.Assert.NoError(err)

	assetsToBalance := map[string]xdr.Int64{}
	for i, symbol := range assets {
		assetsToBalance[symbol.String()] = balances[i]
	}

	expected := map[string]xdr.Int64{
		"credit_alphanum4/BRL/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H": 1000,
		"credit_alphanum4/EUR/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H": 30000,
		"native": 20000,
	}

	tt.Assert.Equal(expected, assetsToBalance)
}
