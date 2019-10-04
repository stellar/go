package history

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

type sqlResult struct {
	lastInsertId int64
	rowsAffected int64
}

func (r sqlResult) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

func (r sqlResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

func TestInsertTrustLine(t *testing.T) {
	mockSession := &db.MockSession{}
	q := &Q{mockSession}

	const expectedQuery = "INSERT INTO trust_lines (accountid,assetcode,assetissuer,assettype,balance,buyingliabilities,flags,last_modified_ledger,lkey,sellingliabilities,tlimit) VALUES (?,?,?,?,?,?,?,?,?,?,?)"

	mockSession.
		On("Exec", mock.Anything).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).(squirrel.Sqlizer)
			q, qargs, err := arg.ToSql()
			assert.Equal(t, expectedQuery, q)
			assert.Equal(t, []interface{}{
				"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
				"EUR",
				"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
				xdr.AssetTypeAssetTypeCreditAlphanum4, // assettype
				xdr.Int64(20000),                      // balance
				xdr.Int64(3),                          // buyinliabilities
				xdr.Uint32(1),                         // flags
				xdr.Uint32(1234),                      // last_modified_ledger
				// lkey:
				"AAAAAQAAAAAdBJqAD9qPq+j2nRDdjdp5KVoUh8riPkNO9ato7BNs8wAAAAFFVVIAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3",
				xdr.Int64(4),         // sellingliabilities
				xdr.Int64(223456789), // tlimit
			}, qargs)
			assert.NoError(t, err)
		}).
		Return(sqlResult{rowsAffected: 1}, nil).
		Once()
	rows, err := q.InsertTrustLine(eurTrustLine, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	mockSession.
		On("Exec", mock.Anything).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).(squirrel.Sqlizer)
			q, qargs, err := arg.ToSql()
			assert.Equal(t, expectedQuery, q)
			assert.Equal(t, []interface{}{
				"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
				"USDUSD",
				"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
				xdr.AssetTypeAssetTypeCreditAlphanum12, // assettype
				xdr.Int64(10000),                       // balance
				xdr.Int64(1),                           // buyinliabilities
				xdr.Uint32(0),                          // flags
				xdr.Uint32(1235),                       // last_modified_ledger
				// lkey:
				"AAAAAQAAAAAdBJqAD9qPq+j2nRDdjdp5KVoUh8riPkNO9ato7BNs8wAAAAJVU0RVU0QAAAAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8Bfc=",
				xdr.Int64(2),         // sellingliabilities
				xdr.Int64(123456789), // tlimit
			}, qargs)
			assert.NoError(t, err)
		}).
		Return(sqlResult{rowsAffected: 1}, nil).
		Once()
	rows, err = q.InsertTrustLine(usdTrustLine, 1235)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)
}

func TestUpdateTrustLine(t *testing.T) {
	mockSession := &db.MockSession{}
	q := &Q{mockSession}

	const expectedQuery = "UPDATE trust_lines SET accountid = ?, assetcode = ?, assetissuer = ?, assettype = ?, balance = ?, buyingliabilities = ?, flags = ?, last_modified_ledger = ?, sellingliabilities = ?, tlimit = ? WHERE lkey = ?"

	mockSession.
		On("Exec", mock.Anything).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).(squirrel.Sqlizer)
			q, qargs, err := arg.ToSql()
			assert.Equal(t, expectedQuery, q)
			assert.Equal(t, []interface{}{
				"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
				"EUR",
				"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
				xdr.AssetTypeAssetTypeCreditAlphanum4, // assettype
				xdr.Int64(20000),                      // balance
				xdr.Int64(3),                          // buyinliabilities
				xdr.Uint32(1),                         // flags
				xdr.Uint32(1234),                      // last_modified_ledger
				xdr.Int64(4),                          // sellingliabilities
				xdr.Int64(223456789),                  // tlimit
				// lkey:
				"AAAAAQAAAAAdBJqAD9qPq+j2nRDdjdp5KVoUh8riPkNO9ato7BNs8wAAAAFFVVIAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3",
			}, qargs)
			assert.NoError(t, err)
		}).
		Return(sqlResult{rowsAffected: 1}, nil).
		Once()
	rows, err := q.UpdateTrustLine(eurTrustLine, 1234)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)
}

func TestRemoveTrustLine(t *testing.T) {
	mockSession := &db.MockSession{}
	q := &Q{mockSession}

	ledgerKeyTrustline := xdr.LedgerKeyTrustLine{
		AccountId: eurTrustLine.AccountId,
		Asset:     eurTrustLine.Asset,
	}

	mockSession.
		On("Exec", mock.Anything).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).(squirrel.Sqlizer)
			q, qargs, err := arg.ToSql()
			assert.Equal(t, "DELETE FROM trust_lines WHERE lkey = ?", q)
			assert.Equal(t, []interface{}{
				// lkey:
				"AAAAAQAAAAAdBJqAD9qPq+j2nRDdjdp5KVoUh8riPkNO9ato7BNs8wAAAAFFVVIAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3",
			}, qargs)
			assert.NoError(t, err)
		}).
		Return(sqlResult{rowsAffected: 1}, nil).
		Once()
	rows, err := q.RemoveTrustLine(ledgerKeyTrustline)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rows)
}
