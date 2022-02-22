package history

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
)

const (
	filterRulesTableName              = "ingest_filter_rules"
	filterRulesTypeColumnName         = "name"
	filterRulesColumnName             = "rules"
	filterRulesEnabledColumnName      = "enabled"
	filterRulesLastModifiedColumnName = "last_modified"
	FilterAssetFilterName             = "asset"
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
    upsertFields := []upsertField{
		{filterRulesLastModifiedColumnName, "bigint", []interface{}{time.Now().Unix()}},
		{filterRulesEnabledColumnName, "bool", []interface{}{config.Enabled}},
		{filterRulesColumnName, "jsonb", []interface{}{config.Rules}},
		{filterRulesTypeColumnName, "text", []interface{}{config.Name}},
	}

	return q.upsertRows(ctx, filterRulesTableName, filterRulesTypeColumnName, upsertFields)
}
