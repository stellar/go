package history

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

const (
	accountFilterWhitelistTableName  = "account_filter_whitelist"
	accountFilterWhitelistColumnName = "account_id"
)

type QAccountFilterWhitelist interface {
	GetAccountFilterWhitelist(ctx context.Context) ([]string, error)
	SetAccountFilterWhitelist(ctx context.Context, accountIDs []string) error
}

// GetAccountFilterWhitelist loads all the entries from account_filter_whitelist
// TODO: the mere existence of this function assumes that the filter's size is small enough for fully querying it into memory
//       We may need to reconsider this if it's not the case.
func (q *Q) GetAccountFilterWhitelist(ctx context.Context) ([]string, error) {
	var accountIDs []string
	sql := sq.Select(accountFilterWhitelistColumnName).From(accountFilterWhitelistTableName)
	err := q.Select(ctx, &accountIDs, sql)
	return accountIDs, err
}

// UpsertAccountFilterWhitelist upserts a batch of data in the account_filter_whitelist table.
func (q *Q) SetAccountFilterWhitelist(ctx context.Context, accountIDs []string) error {
	del := sq.Delete(accountFilterWhitelistTableName)
	// 1. Delete everything on the table
	if _, err := q.Exec(ctx, del); err != nil {
		return err
	}

	if len(accountIDs) == 0 {
		return nil
	}
	// 2. Set the provided fields
	insert := sq.Insert(accountFilterWhitelistTableName).Columns(accountFilterWhitelistColumnName)
	for _, a := range accountIDs {
		insert.Values(a)
	}
	_, err := q.Exec(ctx, insert)
	return err
}

// RemoveFromAccountFilterWhitelist deletes rows from the account_filter_whitelist table.
// Returns the number of rows affected and error.
func (q *Q) RemoveFromAccountFilterWhitelist(ctx context.Context, accountIDs []string) (int64, error) {
	sql := sq.Delete(accountFilterWhitelistTableName).
		Where(sq.Eq{accountFilterWhitelistColumnName: accountIDs})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
