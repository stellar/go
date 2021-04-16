package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var (
	data1 = xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeData,
			Data: &xdr.DataEntry{
				AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
				DataName:  "test data",
				// This also tests if base64 encoding is working as 0 is invalid UTF-8 byte
				DataValue: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
		},
		LastModifiedLedgerSeq: 1234,
	}

	data2 = xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeData,
			Data: &xdr.DataEntry{
				AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
				DataName:  "test data2",
				DataValue: []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			},
		},
		LastModifiedLedgerSeq: 1234,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}
)

func TestInsertAccountData(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccountData(tt.Ctx, data1)
	assert.NoError(t, err)
	tt.Assert.Equal(int64(1), rows)

	rows, err = q.InsertAccountData(tt.Ctx, data2)
	assert.NoError(t, err)
	tt.Assert.Equal(int64(1), rows)

	keys := []xdr.LedgerKeyData{
		{AccountId: data1.Data.Data.AccountId, DataName: data1.Data.Data.DataName},
		{AccountId: data2.Data.Data.AccountId, DataName: data2.Data.Data.DataName},
	}

	datas, err := q.GetAccountDataByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, datas, 2)

	tt.Assert.Equal(data1.Data.Data.DataName, xdr.String64(datas[0].Name))
	tt.Assert.Equal([]byte(data1.Data.Data.DataValue), []byte(datas[0].Value))
	tt.Assert.True(datas[0].Sponsor.IsZero())

	tt.Assert.Equal(data2.Data.Data.DataName, xdr.String64(datas[1].Name))
	tt.Assert.Equal([]byte(data2.Data.Data.DataValue), []byte(datas[1].Value))
	tt.Assert.Equal("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", datas[1].Sponsor.String)
}

func TestUpdateAccountData(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccountData(tt.Ctx, data1)
	assert.NoError(t, err)
	tt.Assert.Equal(int64(1), rows)

	modifiedData := data1
	modifiedData.Data.Data.DataValue[0] = 1

	rows, err = q.UpdateAccountData(tt.Ctx, modifiedData)
	assert.NoError(t, err)
	tt.Assert.Equal(int64(1), rows)

	keys := []xdr.LedgerKeyData{
		{AccountId: data1.Data.Data.AccountId, DataName: data1.Data.Data.DataName},
	}
	datas, err := q.GetAccountDataByKeys(tt.Ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, datas, 1)

	tt.Assert.Equal(modifiedData.Data.Data.DataName, xdr.String64(datas[0].Name))
	tt.Assert.Equal([]byte(modifiedData.Data.Data.DataValue), []byte(datas[0].Value))
	tt.Assert.Equal(uint32(1234), datas[0].LastModifiedLedger)
}

func TestRemoveAccountData(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccountData(tt.Ctx, data1)
	assert.NoError(t, err)
	tt.Assert.Equal(int64(1), rows)

	key := xdr.LedgerKeyData{AccountId: data1.Data.Data.AccountId, DataName: data1.Data.Data.DataName}
	rows, err = q.RemoveAccountData(tt.Ctx, key)
	assert.NoError(t, err)
	tt.Assert.Equal(int64(1), rows)

	datas, err := q.GetAccountDataByKeys(tt.Ctx, []xdr.LedgerKeyData{key})
	assert.NoError(t, err)
	assert.Len(t, datas, 0)

	// Doesn't exist anymore
	rows, err = q.RemoveAccountData(tt.Ctx, key)
	assert.NoError(t, err)
	tt.Assert.Equal(int64(0), rows)
}

func TestGetAccountDataByAccountsID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.InsertAccountData(tt.Ctx, data1)
	assert.NoError(t, err)
	_, err = q.InsertAccountData(tt.Ctx, data2)
	assert.NoError(t, err)

	ids := []string{
		data1.Data.Data.AccountId.Address(),
		data2.Data.Data.AccountId.Address(),
	}
	datas, err := q.GetAccountDataByAccountsID(tt.Ctx, ids)
	assert.NoError(t, err)
	assert.Len(t, datas, 2)

	tt.Assert.Equal(data1.Data.Data.DataName, xdr.String64(datas[0].Name))
	tt.Assert.Equal([]byte(data1.Data.Data.DataValue), []byte(datas[0].Value))

	tt.Assert.Equal(data2.Data.Data.DataName, xdr.String64(datas[1].Name))
	tt.Assert.Equal([]byte(data2.Data.Data.DataValue), []byte(datas[1].Value))
}

func TestGetAccountDataByAccountID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.InsertAccountData(tt.Ctx, data1)
	assert.NoError(t, err)
	_, err = q.InsertAccountData(tt.Ctx, data2)
	assert.NoError(t, err)

	records, err := q.GetAccountDataByAccountID(tt.Ctx, data1.Data.Data.AccountId.Address())
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	tt.Assert.Equal(data1.Data.Data.DataName, xdr.String64(records[0].Name))
	tt.Assert.Equal([]byte(data1.Data.Data.DataValue), []byte(records[0].Value))

	tt.Assert.Equal(data2.Data.Data.DataName, xdr.String64(records[1].Name))
	tt.Assert.Equal([]byte(data2.Data.Data.DataValue), []byte(records[1].Value))
}

func TestGetAccountDataByName(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.InsertAccountData(tt.Ctx, data1)
	assert.NoError(t, err)
	_, err = q.InsertAccountData(tt.Ctx, data2)
	assert.NoError(t, err)

	record, err := q.GetAccountDataByName(tt.Ctx, data1.Data.Data.AccountId.Address(), string(data1.Data.Data.DataName))
	assert.NoError(t, err)
	tt.Assert.Equal(data1.Data.Data.DataName, xdr.String64(record.Name))
	tt.Assert.Equal([]byte(data1.Data.Data.DataValue), []byte(record.Value))

	record, err = q.GetAccountDataByName(tt.Ctx, data1.Data.Data.AccountId.Address(), string(data2.Data.Data.DataName))
	assert.NoError(t, err)
	tt.Assert.Equal(data2.Data.Data.DataName, xdr.String64(record.Name))
	tt.Assert.Equal([]byte(data2.Data.Data.DataValue), []byte(record.Value))

}
