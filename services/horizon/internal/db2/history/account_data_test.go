package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var (
	data1 = xdr.DataEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		DataName:  "test data",
		// This also tests if base64 encoding is working as 0 is invalid UTF-8 byte
		DataValue: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	data2 = xdr.DataEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		DataName:  "test data2",
		DataValue: []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
	}
)

func TestInsertAccountData(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccountData(data1, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	rows, err = q.InsertAccountData(data2, 1235)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	keys := []xdr.LedgerKeyData{
		{AccountId: data1.AccountId, DataName: data1.DataName},
		{AccountId: data2.AccountId, DataName: data2.DataName},
	}

	datas, err := q.GetAccountDataByKeys(keys)
	assert.NoError(t, err)
	assert.Len(t, datas, 2)

	assert.Equal(t, data1.DataName, xdr.String64(datas[0].Name))
	assert.Equal(t, []byte(data1.DataValue), []byte(datas[0].Value))

	assert.Equal(t, data2.DataName, xdr.String64(datas[1].Name))
	assert.Equal(t, []byte(data2.DataValue), []byte(datas[1].Value))
}

func TestUpdateAccountData(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccountData(data1, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	modifiedData := data1
	modifiedData.DataValue[0] = 1

	rows, err = q.UpdateAccountData(modifiedData, 1235)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	keys := []xdr.LedgerKeyData{
		{AccountId: data1.AccountId, DataName: data1.DataName},
	}
	datas, err := q.GetAccountDataByKeys(keys)
	assert.NoError(t, err)
	assert.Len(t, datas, 1)

	assert.Equal(t, modifiedData.DataName, xdr.String64(datas[0].Name))
	assert.Equal(t, []byte(modifiedData.DataValue), []byte(datas[0].Value))
	assert.Equal(t, uint32(1235), datas[0].LastModifiedLedger)
}

func TestRemoveAccountData(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertAccountData(data1, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	key := xdr.LedgerKeyData{AccountId: data1.AccountId, DataName: data1.DataName}
	rows, err = q.RemoveAccountData(key)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	datas, err := q.GetAccountDataByKeys([]xdr.LedgerKeyData{key})
	assert.NoError(t, err)
	assert.Len(t, datas, 0)

	// Doesn't exist anymore
	rows, err = q.RemoveAccountData(key)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}

func TestGetAccountDataByAccountsID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.InsertAccountData(data1, 1234)
	assert.NoError(t, err)
	_, err = q.InsertAccountData(data2, 1235)
	assert.NoError(t, err)

	ids := []string{
		data1.AccountId.Address(),
		data2.AccountId.Address(),
	}
	datas, err := q.GetAccountDataByAccountsID(ids)
	assert.NoError(t, err)
	assert.Len(t, datas, 2)

	assert.Equal(t, data1.DataName, xdr.String64(datas[0].Name))
	assert.Equal(t, []byte(data1.DataValue), []byte(datas[0].Value))

	assert.Equal(t, data2.DataName, xdr.String64(datas[1].Name))
	assert.Equal(t, []byte(data2.DataValue), []byte(datas[1].Value))
}
