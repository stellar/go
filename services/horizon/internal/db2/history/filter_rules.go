package history

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

const (
	filterRulesTableName  = "ingest_filter_rules"
	filterRulesTypeColumnName = "name"
	filterRulesColumnName = "rules"
	filterRulesEnabledColumnName = "enabled"
	filterRulesLastModifiedColumnName = "last_modified"
	FilterAssetFilterName = "asset"
)

type FilterConfig struct {
	Enabled bool        `db:"enabled"`
	Rules string        `db:"rules"`
	Name string         `db:"name"`
	LastModified uint64 `db:"last_modified"`   
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
	sql := sq.Select().From(filterRulesTableName).Where(sq.Eq{filterRulesTypeColumnName: name,})
	err := q.Select(ctx, filterConfig, sql)
    
	return filterConfig, err
}

func (q *Q) SetFilterConfig(ctx context.Context, config FilterConfig) error {
	updateCols := map[string]interface{}{
        filterRulesLastModifiedColumnName: sq.Expr("extract(epoch from now() at time zone 'utc')"),
        filterRulesEnabledColumnName: config.Enabled,
		filterRulesColumnName: config.Rules,
	}
	
	sql := sq.Update(filterRulesTableName).SetMap(updateCols).Where(
		sq.Eq{filterRulesTypeColumnName: config.Name,})
	
	_, err := q.Exec(ctx, sql);
	return err
}
