package history

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

const (
	assetFilterRulesTableName  = "asset_filter_rules"
	assetFilterRulesColumnName = "rules"
)

type QAccountFilterWhitelist interface {
	GetAssetFilterWhitelist(ctx context.Context) ([]string, error)
	GetAssetFilterRules(ctx context.Context) (string, error)
	SetAssetFilterRules(ctx context.Context, filterRules string) error
}

func (q *Q) GetAssetFilterWhitelist(ctx context.Context) ([]string, error) {
	var rules string
	sql := sq.Select(assetFilterRulesColumnName).From(assetFilterRulesTableName)
	err := q.Select(ctx, &rules, sql)
    


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
