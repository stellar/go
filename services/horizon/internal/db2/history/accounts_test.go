package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var (
	inflationDest = xdr.MustAddress("GBUH7T6U36DAVEKECMKN5YEBQYZVRBPNSZAAKBCO6P5HBMDFSQMQL4Z4")

	account1 = xdr.AccountEntry{
		AccountId:     xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Balance:       20000,
		SeqNum:        223456789,
		NumSubEntries: 10,
		InflationDest: &inflationDest,
		Flags:         1,
		HomeDomain:    "stellar.org",
		Thresholds:    xdr.Thresholds{1, 2, 3, 4},
		Ext: xdr.AccountEntryExt{
			V: 1,
			V1: &xdr.AccountEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  3,
					Selling: 4,
				},
			},
		},
	}

	account2 = xdr.AccountEntry{
		AccountId:     xdr.MustAddress("GCT2NQM5KJJEF55NPMY444C6M6CA7T33HRNCMA6ZFBIIXKNCRO6J25K7"),
		Balance:       50000,
		SeqNum:        648736,
		NumSubEntries: 10,
		InflationDest: &inflationDest,
		Flags:         2,
		HomeDomain:    "meridian.stellar.org",
		Thresholds:    xdr.Thresholds{5, 6, 7, 8},
		Ext: xdr.AccountEntryExt{
			V: 1,
			V1: &xdr.AccountEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  30,
					Selling: 40,
				},
			},
		},
	}
)

func TestInsertAccount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccount(account1, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	rows, err = q.InsertAccount(account2, 1235)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	accounts, err := q.GetAccountsByIDs([]string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GCT2NQM5KJJEF55NPMY444C6M6CA7T33HRNCMA6ZFBIIXKNCRO6J25K7",
	})
	assert.NoError(t, err)
	assert.Len(t, accounts, 2)

	assert.Equal(t, "GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB", accounts[0].AccountID)
	assert.Equal(t, int64(20000), accounts[0].Balance)
	assert.Equal(t, int64(223456789), accounts[0].SequenceNumber)
	assert.Equal(t, uint32(10), accounts[0].NumSubEntries)
	assert.Equal(t, "GBUH7T6U36DAVEKECMKN5YEBQYZVRBPNSZAAKBCO6P5HBMDFSQMQL4Z4", accounts[0].InflationDestination)
	assert.Equal(t, uint32(1), accounts[0].Flags)
	assert.Equal(t, "stellar.org", accounts[0].HomeDomain)
	assert.Equal(t, byte(1), accounts[0].MasterWeight)
	assert.Equal(t, byte(2), accounts[0].ThresholdLow)
	assert.Equal(t, byte(3), accounts[0].ThresholdMedium)
	assert.Equal(t, byte(4), accounts[0].ThresholdHigh)
	assert.Equal(t, int64(3), accounts[0].BuyingLiabilities)
	assert.Equal(t, int64(4), accounts[0].SellingLiabilities)
}

func TestUpdateAccount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccount(account1, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	modifiedAccount := account1
	modifiedAccount.Balance = 32847893

	rows, err = q.UpdateAccount(modifiedAccount, 1235)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	keys := []string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GCT2NQM5KJJEF55NPMY444C6M6CA7T33HRNCMA6ZFBIIXKNCRO6J25K7",
	}
	accounts, err := q.GetAccountsByIDs(keys)
	assert.NoError(t, err)
	assert.Len(t, accounts, 1)

	expectedBinary, err := modifiedAccount.MarshalBinary()
	assert.NoError(t, err)

	dbEntry := xdr.AccountEntry{
		AccountId:     xdr.MustAddress(accounts[0].AccountID),
		Balance:       xdr.Int64(accounts[0].Balance),
		SeqNum:        xdr.SequenceNumber(accounts[0].SequenceNumber),
		NumSubEntries: xdr.Uint32(accounts[0].NumSubEntries),
		InflationDest: &inflationDest,
		Flags:         xdr.Uint32(accounts[0].Flags),
		HomeDomain:    xdr.String32(accounts[0].HomeDomain),
		Thresholds: xdr.Thresholds{
			accounts[0].MasterWeight,
			accounts[0].ThresholdLow,
			accounts[0].ThresholdMedium,
			accounts[0].ThresholdHigh,
		},
		Ext: xdr.AccountEntryExt{
			V: 1,
			V1: &xdr.AccountEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  xdr.Int64(accounts[0].BuyingLiabilities),
					Selling: xdr.Int64(accounts[0].SellingLiabilities),
				},
			},
		},
	}

	actualBinary, err := dbEntry.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, expectedBinary, actualBinary)
	assert.Equal(t, uint32(1235), accounts[0].LastModifiedLedger)
}

func TestRemoveAccount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccount(account1, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	rows, err = q.RemoveAccount("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	accounts, err := q.GetAccountsByIDs([]string{"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"})
	assert.NoError(t, err)
	assert.Len(t, accounts, 0)

	// Doesn't exist anymore
	rows, err = q.RemoveAccount("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}

func TestAccountsForAsset(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	eurTrustLine.AccountId = account1.AccountId
	usdTrustLine.AccountId = account2.AccountId

	_, err := q.InsertAccount(account1, 1234)
	tt.Assert.NoError(err)
	_, err = q.InsertAccount(account2, 1235)
	tt.Assert.NoError(err)

	_, err = q.InsertTrustLine(eurTrustLine, 1234)
	tt.Assert.NoError(err)
	_, err = q.InsertTrustLine(usdTrustLine, 1235)
	tt.Assert.NoError(err)

	pq := db2.PageQuery{
		Order:  db2.OrderAscending,
		Limit:  db2.DefaultPageSize,
		Cursor: "",
	}

	accounts, err := q.AccountsForAsset(eurTrustLine.Asset, pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 1)
	tt.Assert.Equal(account1.AccountId.Address(), accounts[0].AccountID)

	accounts, err = q.AccountsForAsset(usdTrustLine.Asset, pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 1)
	tt.Assert.Equal(account2.AccountId.Address(), accounts[0].AccountID)

	pq.Cursor = account2.AccountId.Address()
	accounts, err = q.AccountsForAsset(usdTrustLine.Asset, pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 0)
}
