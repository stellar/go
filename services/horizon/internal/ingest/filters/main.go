package filters

import (
	"context"
	"time"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/log"
)

const (

	// the filter config cache will be checked against latest from db at most once per each of this interval,
	filterConfigCheckIntervalSeconds int64 = 10
)

var (
	LOG = log.WithFields(log.F{
		"filters": "load",
	})
)

type filtersCache struct {
	cachedFilters                  map[string]processors.LedgerTransactionFilterer
	lastFilterConfigCheckUnixEpoch int64
}

type Filters interface {
	GetFilters(filterQ history.QFilter, ctx context.Context) []processors.LedgerTransactionFilterer
}

func NewFilters() Filters {
	return &filtersCache{
		cachedFilters: map[string]processors.LedgerTransactionFilterer{
			horizon.IngestionFilterAssetName:   NewAssetFilter(),
			horizon.IngestionFilterAccountName: NewAccountFilter(),
		},
	}
}

// Provide list of the active filters. Optimize performance by caching the list, only
// rebuild the list on expiration time interval. Method is NOT thread-safe.
func (f *filtersCache) GetFilters(filterQ history.QFilter, ctx context.Context) []processors.LedgerTransactionFilterer {
	// TODO, should we put a mutex/sync on this to be safe? currently not re-entrant,
	// bound to instance of filtersCache, looks like it is only invoked serially per ledger from a processor,
	// thinking can safely avoid the sync overhead?

	// only attempt to refresh filter config cache state at configured interval limit
	if time.Now().Unix() < (f.lastFilterConfigCheckUnixEpoch + filterConfigCheckIntervalSeconds) {
		return f.convertCacheToList()
	}

	f.lastFilterConfigCheckUnixEpoch = time.Now().Unix()

	LOG.Info("expired filter config cache, refresh from db")
	filterConfigs, err := filterQ.GetAllFilters(ctx)
	if err != nil {
		LOG.Errorf("unable to query filter configs, %v", err)
		// allow the error, fall back to last loaded config
		return f.convertCacheToList()
	}

	for _, filterConfig := range filterConfigs {
		switch filterConfig.Name {
		case horizon.IngestionFilterAssetName:
			assetFilter := f.cachedFilters[horizon.IngestionFilterAssetName].(AssetFilter)
			err := assetFilter.RefreshAssetFilter(&filterConfig)
			if err != nil {
				LOG.Errorf("unable to refresh asset filter config %v", err)
				continue
			}
		case horizon.IngestionFilterAccountName:
			accountFilter := f.cachedFilters[horizon.IngestionFilterAccountName].(AccountFilter)
			err := accountFilter.RefreshAccountFilter(&filterConfig)
			if err != nil {
				LOG.Errorf("unable to refresh account filter config %v", err)
				continue
			}
		}
	}
	return f.convertCacheToList()
}

func (f *filtersCache) convertCacheToList() []processors.LedgerTransactionFilterer {
	filters := []processors.LedgerTransactionFilterer{}
	for _, filter := range f.cachedFilters {
		filters = append(filters, filter)
	}
	return filters
}
