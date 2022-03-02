package history

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

const (
	filterRulesTableName              = "ingest_filter_rules"
	filterRulesTypeColumnName         = "name"
	filterRulesColumnName             = "rules"
	filterRulesEnabledColumnName      = "enabled"
	filterRulesLastModifiedColumnName = "last_modified"
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
	UpdateFilterConfig(ctx context.Context, config FilterConfig) error
}

func (q *Q) GetAllFilters(ctx context.Context) ([]FilterConfig, error) {
	var filterConfigs []FilterConfig
	sql := sq.Select("*").From(filterRulesTableName)
	err := q.Select(ctx, &filterConfigs, sql)

	return filterConfigs, err
}

func (q *Q) GetFilterByName(ctx context.Context, name string) (FilterConfig, error) {
	filterConfig := FilterConfig{}
	sql := sq.Select("*").From(filterRulesTableName).Where(sq.Eq{filterRulesTypeColumnName: name})
	err := q.Get(ctx, &filterConfig, sql)

	return filterConfig, err
}

func (q *Q) UpdateFilterConfig(ctx context.Context, config FilterConfig) error {
	updateCols := map[string]interface{}{
		filterRulesLastModifiedColumnName: sq.Expr(`extract(epoch from now() at time zone 'utc')`),
		filterRulesEnabledColumnName:      config.Enabled,
		filterRulesColumnName:             sq.Expr(fmt.Sprintf(`'%v'::json`, config.Rules)),
		filterRulesTypeColumnName:         config.Name,
	}

	sqlUpdate := sq.Update(filterRulesTableName).SetMap(updateCols).Where(
		sq.Eq{filterRulesTypeColumnName: config.Name})

	rowCnt, err := q.checkForError(sqlUpdate, ctx)
	if err != nil {
		return err
	}

	if rowCnt < 1 {
		return sql.ErrNoRows
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
