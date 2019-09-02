package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/errors"
)

func (q *Q) CountAccounts() (int, error) {
	sql := sq.Select("count(distinct account)").From("accounts_signers")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

func (q *Q) SignersForAccounts(accounts []string) ([]AccountSigner, error) {
	sql := selectAccountSigners.Where(map[string]interface{}{"accounts_signers.account": accounts})

	var results []AccountSigner
	if err := q.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

// AccountsForSigner returns a list of `AccountSigner` rows for a given signer
func (q *Q) AccountsForSigner(signer string, page db2.PageQuery) ([]AccountSigner, error) {
	sql := selectAccountSigners.Where("accounts_signers.signer = ?", signer)
	sql, err := page.ApplyToUsingCursor(sql, "accounts_signers.account", page.Cursor)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	var results []AccountSigner
	if err := q.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

// InsertAccountSigner creates a row in the accounts_signers table
func (q *Q) InsertAccountSigner(account, signer string, weight int32) error {
	sql := sq.Insert("accounts_signers").
		Columns("account", "signer", "weight").
		Values(account, signer, weight)
	_, err := q.Exec(sql)
	return err
}

// UpsertAccountSigner creates a row in the accounts_signers table
func (q *Q) UpsertAccountSigner(account, signer string, weight int32) error {
	sql := sq.Insert("accounts_signers").
		Columns("account", "signer", "weight").
		Values(account, signer, weight).
		Suffix("ON CONFLICT (signer, account) DO UPDATE SET weight=EXCLUDED.weight")

	_, err := q.Exec(sql)
	return err
}

// RemoveAccountSigner deletes a row in the accounts_signers table
func (q *Q) RemoveAccountSigner(account, signer string) error {
	sql := sq.Delete("accounts_signers").Where(sq.Eq{
		"account": account,
		"signer":  signer,
	})

	_, err := q.Exec(sql)
	return err
}

var selectAccountSigners = sq.Select("accounts_signers.*").From("accounts_signers")
