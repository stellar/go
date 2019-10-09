package history

import (
	"encoding/base64"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (q *Q) CountAccountsData() (int, error) {
	sql := sq.Select("count(*)").From("accounts_data")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetAccountDataByKeys loads a row from the `accounts_data` table, selected by multiple keys.
func (q *Q) GetAccountDataByKeys(keys []xdr.LedgerKeyData) ([]Data, error) {
	var data []Data
	lkeys := make([]string, 0, len(keys))
	for _, key := range keys {
		lkey, err := ledgerKeyDataToString(key)
		if err != nil {
			return nil, errors.Wrap(err, "Error running ledgerKeyTrustLineToString")
		}
		lkeys = append(lkeys, lkey)
	}
	sql := selectAccountData.Where(map[string]interface{}{"accounts_data.lkey": lkeys})
	err := q.Select(&data, sql)
	return data, err
}

func ledgerKeyDataToString(data xdr.LedgerKeyData) (string, error) {
	ledgerKey := &xdr.LedgerKey{}
	err := ledgerKey.SetData(data.AccountId, string(data.DataName))
	if err != nil {
		return "", errors.Wrap(err, "Error running ledgerKey.SetData")
	}
	key, err := ledgerKey.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Error running MarshalBinaryCompress")
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func dataEntryToLedgerKeyString(data xdr.DataEntry) (string, error) {
	ledgerKey := &xdr.LedgerKey{}
	err := ledgerKey.SetData(data.AccountId, string(data.DataName))
	if err != nil {
		return "", errors.Wrap(err, "Error running ledgerKey.SetTrustline")
	}
	key, err := ledgerKey.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Error running MarshalBinaryCompress")
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

// InsertAccountData creates a row in the accounts_data table.
// Returns number of rows affected and error.
func (q *Q) InsertAccountData(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	// Add lkey only when inserting rows
	key, err := dataEntryToLedgerKeyString(data)
	if err != nil {
		return 0, errors.Wrap(err, "Error running dataEntryToLedgerKeyString")
	}

	sql := sq.Insert("accounts_data").
		Columns("lkey", "account", "name", "value", "last_modified_ledger").
		Values(
			key,
			data.AccountId.Address(),
			data.DataName,
			AccountDataValue(data.DataValue),
			lastModifiedLedger,
		)

	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// UpdateAccountData updates a row in the accounts_data table.
// Returns number of rows affected and error.
func (q *Q) UpdateAccountData(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	key, err := dataEntryToLedgerKeyString(data)
	if err != nil {
		return 0, errors.Wrap(err, "Error running dataEntryToLedgerKeyString")
	}

	sql := sq.Update("accounts_data").
		SetMap(map[string]interface{}{
			"value":                AccountDataValue(data.DataValue),
			"last_modified_ledger": lastModifiedLedger,
		}).
		Where(sq.Eq{"lkey": key})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveAccountData deletes a row in the accounts_data table.
// Returns number of rows affected and error.
func (q *Q) RemoveAccountData(key xdr.LedgerKeyData) (int64, error) {
	lkey, err := ledgerKeyDataToString(key)
	if err != nil {
		return 0, errors.Wrap(err, "Error running ledgerKeyDataToString")
	}

	sql := sq.Delete("accounts_data").
		Where(sq.Eq{"lkey": lkey})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

var selectAccountData = sq.Select(`
	account,
	name,
	value,
	last_modified_ledger
`).From("accounts_data")
