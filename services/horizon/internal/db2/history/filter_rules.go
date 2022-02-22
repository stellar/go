package history

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
)

const (
	filterRulesTableName              = "ingest_filter_rules"
	filterRulesTypeColumnName         = "name"
	filterRulesColumnName             = "rules"
	filterRulesEnabledColumnName      = "enabled"
	filterRulesLastModifiedColumnName = "last_modified"
	FilterAssetFilterName             = "asset"
	FilterAccountFilterName           = "account"
)

type FilterConfig struct {
	Enabled      bool   `db:"enabled"`
	Rules        string `db:"rules"`
	Name         string `db:"name"`
	LastModified int64  `db:"last_modified"`
}

type QFilter interface {
	GetAllFilters(ctx context.Context) ([]FilterConfig, error)
	GetFilterByName(ctx context.Context, name string) (FilterConfig, error)
	SetFilterConfig(ctx context.Context, config FilterConfig) error
}

func (q *Q) GetAllFilters(ctx context.Context) ([]FilterConfig, error) {
	var filterConfigs []FilterConfig
	sql := sq.Select().From(filterRulesTableName)
	err := q.Select(ctx, filterConfigs, sql)

	return filterConfigs, err
}

func (q *Q) GetFilterByName(ctx context.Context, name string) (FilterConfig, error) {
	var filterConfig FilterConfig
	sql := sq.Select().From(filterRulesTableName).Where(sq.Eq{filterRulesTypeColumnName: name})
	err := q.Select(ctx, filterConfig, sql)

	return filterConfig, err
}

func (q *Q) SetFilterConfig(ctx context.Context, config FilterConfig) error {
	updateCols := map[string]interface{}{
		filterRulesLastModifiedColumnName: sq.Expr("extract(epoch from now() at time zone 'utc')"),
		filterRulesEnabledColumnName:      config.Enabled,
		filterRulesColumnName:             config.Rules,
		filterRulesTypeColumnName:         config.Name,
	}

	sqlUpdate := sq.Update(filterRulesTableName).SetMap(updateCols).Where(
		sq.Eq{filterRulesTypeColumnName: config.Name})

	rowCnt, err := q.checkForError(sqlUpdate, ctx)
	if err != nil {
		return err
	}

	if rowCnt < 1 {
		sqlInsert := sq.Insert(filterRulesTableName).SetMap(updateCols)
		rowCnt, err = q.checkForError(sqlInsert, ctx)
		if err != nil {
			return err
		}
		if rowCnt < 1 {
			return errors.Errorf("insertion of filter rule did not result in new row created in db")
		}
	}
	return nil
}

func (q *Q) checkForError(builder sq.Sqlizer, ctx context.Context) (int64, error) {
	result, err := q.Exec(ctx, builder)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
