package history

import (
	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// IsAuthRequired returns true if the account has the "AUTH_REQUIRED" option
// turned on.
func (account AccountEntry) IsAuthRequired() bool {
	return xdr.AccountFlags(account.Flags).IsAuthRequired()
}

// IsAuthRevocable returns true if the account has the "AUTH_REVOCABLE" option
// turned on.
func (account AccountEntry) IsAuthRevocable() bool {
	return xdr.AccountFlags(account.Flags).IsAuthRevocable()
}

// IsAuthImmutable returns true if the account has the "AUTH_IMMUTABLE" option
// turned on.
func (account AccountEntry) IsAuthImmutable() bool {
	return xdr.AccountFlags(account.Flags).IsAuthImmutable()
}

func (q *Q) CountAccounts() (int, error) {
	sql := sq.Select("count(*)").From("accounts")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

func (q *Q) GetAccountByID(id string) (AccountEntry, error) {
	var account AccountEntry
	sql := selectAccounts.Where(sq.Eq{"account_id": id})
	err := q.Get(&account, sql)
	return account, err
}

func (q *Q) GetAccountsByIDs(ids []string) ([]AccountEntry, error) {
	var accounts []AccountEntry
	sql := selectAccounts.Where(map[string]interface{}{"accounts.account_id": ids})
	err := q.Select(&accounts, sql)
	return accounts, err
}

func accountToMap(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) map[string]interface{} {
	var buyingliabilities, sellingliabilities xdr.Int64
	if account.Ext.V1 != nil {
		v1 := account.Ext.V1
		buyingliabilities = v1.Liabilities.Buying
		sellingliabilities = v1.Liabilities.Selling
	}

	var inflationDestination = ""
	if account.InflationDest != nil {
		inflationDestination = account.InflationDest.Address()
	}

	return map[string]interface{}{
		"account_id":            account.AccountId.Address(),
		"balance":               account.Balance,
		"buying_liabilities":    buyingliabilities,
		"selling_liabilities":   sellingliabilities,
		"sequence_number":       account.SeqNum,
		"num_subentries":        account.NumSubEntries,
		"inflation_destination": inflationDestination,
		"flags":                 account.Flags,
		"home_domain":           account.HomeDomain,
		"master_weight":         account.MasterKeyWeight(),
		"threshold_low":         account.ThresholdLow(),
		"threshold_medium":      account.ThresholdMedium(),
		"threshold_high":        account.ThresholdHigh(),
		"last_modified_ledger":  lastModifiedLedger,
	}
}

// InsertAccount creates a row in the accounts table.
// Returns number of rows affected and error.
func (q *Q) InsertAccount(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	m := accountToMap(account, lastModifiedLedger)

	sql := sq.Insert("accounts").SetMap(m)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// UpdateAccount updates a row in the offers table.
// Returns number of rows affected and error.
func (q *Q) UpdateAccount(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	m := accountToMap(account, lastModifiedLedger)

	accountID := m["account_id"]
	delete(m, "account_id")

	sql := sq.Update("accounts").SetMap(m).Where(sq.Eq{"account_id": accountID})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveAccount deletes a row in the offers table.
// Returns number of rows affected and error.
func (q *Q) RemoveAccount(accountID string) (int64, error) {
	sql := sq.Delete("accounts").Where(sq.Eq{"account_id": accountID})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// AccountsForAsset returns a list of `AccountEntry` rows who are trustee to an
// asset
func (q *Q) AccountsForAsset(asset xdr.Asset, page db2.PageQuery) ([]AccountEntry, error) {
	var assetType, code, issuer string
	asset.MustExtract(&assetType, &code, &issuer)

	sql := sq.
		Select("accounts.*").
		From("accounts").
		Join("trust_lines ON accounts.account_id = trust_lines.account_id").
		Where(map[string]interface{}{
			"trust_lines.asset_type":   int32(asset.Type),
			"trust_lines.asset_issuer": issuer,
			"trust_lines.asset_code":   code,
		})

	sql, err := page.ApplyToUsingCursor(sql, "trust_lines.account_id", page.Cursor)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	var results []AccountEntry
	if err := q.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

// AccountEntriesForSigner returns a list of `AccountEntry` rows for a given signer
func (q *Q) AccountEntriesForSigner(signer string, page db2.PageQuery) ([]AccountEntry, error) {
	sql := sq.
		Select("accounts.*").
		From("accounts").
		Join("accounts_signers ON accounts.account_id = accounts_signers.account_id").
		Where(map[string]interface{}{
			"accounts_signers.signer": signer,
		})

	sql, err := page.ApplyToUsingCursor(sql, "accounts_signers.account_id", page.Cursor)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	var results []AccountEntry
	if err := q.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

var selectAccounts = sq.Select(`
	account_id,
	balance,
	buying_liabilities,
	selling_liabilities,
	sequence_number,
	num_subentries,
	inflation_destination,
	flags,
	home_domain,
	master_weight,
	threshold_low,
	threshold_medium,
	threshold_high,
	last_modified_ledger
`).From("accounts")
