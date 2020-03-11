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

	account3 = xdr.AccountEntry{
		AccountId:     xdr.MustAddress("GDPGOMFSP4IF7A4P7UBKA4UC4QTRLEHGBD6IMDIS3W3KBDNBFAQ7FXDY"),
		Balance:       50000,
		SeqNum:        648736,
		NumSubEntries: 10,
		InflationDest: &inflationDest,
		Flags:         2,
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

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(account1, 1234)
	assert.NoError(t, err)
	err = batch.Add(account2, 1235)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec())

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

func TestUpsertAccount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	ledgerEntries := []xdr.LedgerEntry{
		xdr.LedgerEntry{
			LastModifiedLedgerSeq: 1234,
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &account1,
			},
		},
		xdr.LedgerEntry{
			LastModifiedLedgerSeq: 1234,
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &account2,
			},
		},
	}
	err := q.UpsertAccounts(ledgerEntries)
	assert.NoError(t, err)

	modifiedAccount := account1
	modifiedAccount.Balance = 32847893

	err = q.UpsertAccounts([]xdr.LedgerEntry{{
		LastModifiedLedgerSeq: 1235,
		Data: xdr.LedgerEntryData{
			Type:    xdr.LedgerEntryTypeAccount,
			Account: &modifiedAccount,
		},
	}})
	assert.NoError(t, err)

	keys := []string{
		account1.AccountId.Address(),
		account2.AccountId.Address(),
	}
	accounts, err := q.GetAccountsByIDs(keys)
	assert.NoError(t, err)
	assert.Len(t, accounts, 2)

	accounts, err = q.GetAccountsByIDs([]string{account1.AccountId.Address()})
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

	accounts, err = q.GetAccountsByIDs([]string{account2.AccountId.Address()})
	assert.NoError(t, err)
	assert.Len(t, accounts, 1)

	expectedBinary, err = account2.MarshalBinary()
	assert.NoError(t, err)

	dbEntry = xdr.AccountEntry{
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

	actualBinary, err = dbEntry.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, expectedBinary, actualBinary)
	assert.Equal(t, uint32(1234), accounts[0].LastModifiedLedger)
}

func TestRemoveAccount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(account1, 1234)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec())

	var rows int64
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

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(account1, 1234)
	assert.NoError(t, err)
	err = batch.Add(account2, 1235)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec())

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

	pq = db2.PageQuery{
		Order:  db2.OrderDescending,
		Limit:  db2.DefaultPageSize,
		Cursor: "",
	}

	accounts, err = q.AccountsForAsset(eurTrustLine.Asset, pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 1)
}

func TestAccountEntriesForSigner(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	eurTrustLine.AccountId = account1.AccountId
	usdTrustLine.AccountId = account2.AccountId

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(account1, 1234)
	assert.NoError(t, err)
	err = batch.Add(account2, 1235)
	assert.NoError(t, err)
	err = batch.Add(account3, 1235)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec())

	_, err = q.InsertTrustLine(eurTrustLine, 1234)
	tt.Assert.NoError(err)
	_, err = q.InsertTrustLine(usdTrustLine, 1235)
	tt.Assert.NoError(err)

	_, err = q.CreateAccountSigner(account1.AccountId.Address(), account1.AccountId.Address(), 1)
	tt.Assert.NoError(err)
	_, err = q.CreateAccountSigner(account2.AccountId.Address(), account2.AccountId.Address(), 1)
	tt.Assert.NoError(err)
	_, err = q.CreateAccountSigner(account3.AccountId.Address(), account3.AccountId.Address(), 1)
	tt.Assert.NoError(err)
	_, err = q.CreateAccountSigner(account1.AccountId.Address(), account3.AccountId.Address(), 1)
	tt.Assert.NoError(err)
	_, err = q.CreateAccountSigner(account2.AccountId.Address(), account3.AccountId.Address(), 1)
	tt.Assert.NoError(err)

	pq := db2.PageQuery{
		Order:  db2.OrderAscending,
		Limit:  db2.DefaultPageSize,
		Cursor: "",
	}

	accounts, err := q.AccountEntriesForSigner(account1.AccountId.Address(), pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 1)
	tt.Assert.Equal(account1.AccountId.Address(), accounts[0].AccountID)

	accounts, err = q.AccountEntriesForSigner(account2.AccountId.Address(), pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 1)
	tt.Assert.Equal(account2.AccountId.Address(), accounts[0].AccountID)

	want := map[string]bool{
		account1.AccountId.Address(): true,
		account2.AccountId.Address(): true,
		account3.AccountId.Address(): true,
	}

	accounts, err = q.AccountEntriesForSigner(account3.AccountId.Address(), pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 3)

	for _, account := range accounts {
		tt.Assert.True(want[account.AccountID])
		delete(want, account.AccountID)
	}

	tt.Assert.Len(want, 0)

	pq.Cursor = accounts[len(accounts)-1].AccountID
	accounts, err = q.AccountEntriesForSigner(account3.AccountId.Address(), pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 0)

	pq.Order = "desc"
	accounts, err = q.AccountEntriesForSigner(account3.AccountId.Address(), pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 2)

	pq = db2.PageQuery{
		Order:  db2.OrderDescending,
		Limit:  db2.DefaultPageSize,
		Cursor: "",
	}

	accounts, err = q.AccountEntriesForSigner(account1.AccountId.Address(), pq)
	assert.NoError(t, err)
	tt.Assert.Len(accounts, 1)
}

func TestGetAccountByID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(account1, 1234)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec())

	resultAccount, err := q.GetAccountByID(account1.AccountId.Address())
	assert.NoError(t, err)

	assert.Equal(t, "GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB", resultAccount.AccountID)
	assert.Equal(t, int64(20000), resultAccount.Balance)
	assert.Equal(t, int64(223456789), resultAccount.SequenceNumber)
	assert.Equal(t, uint32(10), resultAccount.NumSubEntries)
	assert.Equal(t, "GBUH7T6U36DAVEKECMKN5YEBQYZVRBPNSZAAKBCO6P5HBMDFSQMQL4Z4", resultAccount.InflationDestination)
	assert.Equal(t, uint32(1), resultAccount.Flags)
	assert.Equal(t, "stellar.org", resultAccount.HomeDomain)
	assert.Equal(t, byte(1), resultAccount.MasterWeight)
	assert.Equal(t, byte(2), resultAccount.ThresholdLow)
	assert.Equal(t, byte(3), resultAccount.ThresholdMedium)
	assert.Equal(t, byte(4), resultAccount.ThresholdHigh)
	assert.Equal(t, int64(3), resultAccount.BuyingLiabilities)
	assert.Equal(t, int64(4), resultAccount.SellingLiabilities)
}
