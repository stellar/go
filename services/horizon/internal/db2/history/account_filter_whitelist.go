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
	UpsertAccountFilterWhitelist(ctx context.Context, accountIDs []string) error
	RemoveFromAccountFilterWhitelist(ctx context.Context, accountIDs []string) (int64, error)
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
func (q *Q) UpsertAccountFilterWhitelist(ctx context.Context, accountIDs []string) error {
	var accountID []interface{}

	for _, id := range accountIDs {
		accountID = append(accountID, id)
	}

	upsertFields := []upsertField{
		{accountFilterWhitelistColumnName, "character varying(56)", accountID},
	}

	return q.upsertRows(ctx, accountFilterWhitelistTableName, accountFilterWhitelistColumnName, upsertFields)
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
