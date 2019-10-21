package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var (
	trustLineIssuer = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")

	eurTrustLine = xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   20000,
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
	}

	usdTrustLine = xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
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

	rows, err := q.InsertTrustLine(eurTrustLine, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	rows, err = q.InsertTrustLine(usdTrustLine, 1235)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	keys := []xdr.LedgerKeyTrustLine{
		{Asset: eurTrustLine.Asset, AccountId: eurTrustLine.AccountId},
		{Asset: usdTrustLine.Asset, AccountId: usdTrustLine.AccountId},
	}

	lines, err := q.GetTrustLinesByKeys(keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 2)
}

func TestUpdateTrustLine(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertTrustLine(eurTrustLine, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	modifiedTrustLine := eurTrustLine
	modifiedTrustLine.Balance = 30000

	rows, err = q.UpdateTrustLine(modifiedTrustLine, 1235)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	keys := []xdr.LedgerKeyTrustLine{
		{Asset: eurTrustLine.Asset, AccountId: eurTrustLine.AccountId},
	}
	lines, err := q.GetTrustLinesByKeys(keys)
	assert.NoError(t, err)
	assert.Len(t, lines, 1)

	expectedBinary, err := modifiedTrustLine.MarshalBinary()
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
	assert.Equal(t, uint32(1235), lines[0].LastModifiedLedger)
}

func TestRemoveTrustLine(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	rows, err := q.InsertTrustLine(eurTrustLine, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	key := xdr.LedgerKeyTrustLine{Asset: eurTrustLine.Asset, AccountId: eurTrustLine.AccountId}
	rows, err = q.RemoveTrustLine(key)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	lines, err := q.GetTrustLinesByKeys([]xdr.LedgerKeyTrustLine{key})
	assert.NoError(t, err)
	assert.Len(t, lines, 0)

	// Doesn't exist anymore
	rows, err = q.RemoveTrustLine(key)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}
