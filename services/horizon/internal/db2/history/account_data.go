package history

import (
	"context"
	"encoding/base64"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (q *Q) CountAccountsData(ctx context.Context) (int, error) {
	sql := sq.Select("count(*)").From("accounts_data")

	var count int
	if err := q.Get(ctx, &count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetAccountDataByName loads account data for a given account ID and data name
func (q *Q) GetAccountDataByName(ctx context.Context, id, name string) (Data, error) {
	var data Data
	sql := selectAccountData.Where(sq.Eq{
		"account_id": id,
		"name":       name,
	}).Limit(1)
	err := q.Get(ctx, &data, sql)
	return data, err
}

// GetAccountDataByAccountID loads account data for a given account ID
func (q *Q) GetAccountDataByAccountID(ctx context.Context, id string) ([]Data, error) {
	var data []Data
	sql := selectAccountData.Where(sq.Eq{"account_id": id})
	err := q.Select(ctx, &data, sql)
	return data, err
}

// GetAccountDataByKeys loads a row from the `accounts_data` table, selected by multiple keys.
func (q *Q) GetAccountDataByKeys(ctx context.Context, keys []xdr.LedgerKeyData) ([]Data, error) {
	var data []Data
	lkeys := make([]string, 0, len(keys))
	for _, key := range keys {
		lkey, err := ledgerKeyDataToString(key)
		if err != nil {
			return nil, errors.Wrap(err, "Error running ledgerKeyTrustLineToString")
		}
		lkeys = append(lkeys, lkey)
	}
	sql := selectAccountData.Where(map[string]interface{}{"accounts_data.ledger_key": lkeys})
	err := q.Select(ctx, &data, sql)
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

func dataEntryToLedgerKeyString(entry xdr.LedgerEntry) (string, error) {
	ledgerKey := entry.LedgerKey()
	key, err := ledgerKey.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Error running MarshalBinaryCompress")
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

// InsertAccountData creates a row in the accounts_data table.
// Returns number of rows affected and error.
func (q *Q) InsertAccountData(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	data := entry.Data.MustData()

	// Add lkey only when inserting rows
	key, err := dataEntryToLedgerKeyString(entry)
	if err != nil {
		return 0, errors.Wrap(err, "Error running dataEntryToLedgerKeyString")
	}

	sql := sq.Insert("accounts_data").
		Columns("ledger_key", "account_id", "name", "value", "last_modified_ledger", "sponsor").
		Values(
			key,
			data.AccountId.Address(),
			data.DataName,
			AccountDataValue(data.DataValue),
			entry.LastModifiedLedgerSeq,
			ledgerEntrySponsorToNullString(entry),
		)

	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// UpdateAccountData updates a row in the accounts_data table.
// Returns number of rows affected and error.
func (q *Q) UpdateAccountData(ctx context.Context, entry xdr.LedgerEntry) (int64, error) {
	data := entry.Data.MustData()

	key, err := dataEntryToLedgerKeyString(entry)
	if err != nil {
		return 0, errors.Wrap(err, "Error running dataEntryToLedgerKeyString")
	}
	sql := sq.Update("accounts_data").
		SetMap(map[string]interface{}{
			"value":                AccountDataValue(data.DataValue),
			"last_modified_ledger": entry.LastModifiedLedgerSeq,
			"sponsor":              ledgerEntrySponsorToNullString(entry),
		}).
		Where(sq.Eq{"ledger_key": key})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveAccountData deletes a row in the accounts_data table.
// Returns number of rows affected and error.
func (q *Q) RemoveAccountData(ctx context.Context, key xdr.LedgerKeyData) (int64, error) {
	lkey, err := ledgerKeyDataToString(key)
	if err != nil {
		return 0, errors.Wrap(err, "Error running ledgerKeyDataToString")
	}

	sql := sq.Delete("accounts_data").
		Where(sq.Eq{"ledger_key": lkey})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetAccountDataByAccountsID loads account data for a list of account ID
func (q *Q) GetAccountDataByAccountsID(ctx context.Context, id []string) ([]Data, error) {
	var data []Data
	sql := selectAccountData.Where(sq.Eq{"account_id": id})
	err := q.Select(ctx, &data, sql)
	return data, err
}

var selectAccountData = sq.Select(`
	account_id,
	name,
	value,
	last_modified_ledger,
	sponsor
`).From("accounts_data")
