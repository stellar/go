package filters

import (
	"context"
	"time"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/log"
)

const (
	FilterAssetFilterName   = "asset"
	FilterAccountFilterName = "account"
	// the filter config cache will be checked against latest from db at most once per each of this interval,
	filterConfigCheckIntervalSeconds int64 = 10
)

var (
	supportedFilterNames = []string{FilterAssetFilterName, FilterAccountFilterName}
	LOG                  = log.WithFields(log.F{
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
			FilterAssetFilterName:   NewAccountFilter(),
			FilterAccountFilterName: NewAssetFilter(),
		},
	}
}

// Provide list of the active filters. Optimize performance by caching the list, only
// rebuild the list on expiration time interval. Method is thread-safe.
func (f *filtersCache) GetFilters(filterQ history.QFilter, ctx context.Context) []processors.LedgerTransactionFilterer {
	// only attempt to refresh filter config cache state at configured interval limit
	if time.Now().Unix() < (f.lastFilterConfigCheckUnixEpoch + filterConfigCheckIntervalSeconds) {
		return f.convertCacheToList()
	}

	LOG.Info("expired filter config cache, refresh from db")
	filterConfigs, err := filterQ.GetAllFilters(ctx)
	if err != nil {
		LOG.Errorf("unable to query filter configs, %v", err)
		// reset the cache time regardless, so next attempt is at next interval
		f.lastFilterConfigCheckUnixEpoch = time.Now().Unix()
		return f.convertCacheToList()
	}

	for _, filterConfig := range filterConfigs {
		if filterConfig.Enabled {
			switch filterConfig.Name {
			case FilterAssetFilterName:
				assetFilter := f.cachedFilters[FilterAssetFilterName].(AssetFilter)
				err := assetFilter.RefreshAssetFilter(&filterConfig)
				if err != nil {
					LOG.Errorf("unable to refresh asset filter config %v", err)
					continue
				}
			case FilterAccountFilterName:
				accountFilter := f.cachedFilters[FilterAccountFilterName].(AccountFilter)
				err := accountFilter.RefreshAccountFilter(&filterConfig)
				if err != nil {
					LOG.Errorf("unable to refresh account filter config %v", err)
					continue
				}
			}

		}
	}
	return f.convertCacheToList()
}

func SupportedFilterNames(name string) bool {
	for _, supportedName := range supportedFilterNames {
		if name == supportedName {
			return true
		}
	}
	return false
}

func (f *filtersCache) convertCacheToList() []processors.LedgerTransactionFilterer {
	filters := []processors.LedgerTransactionFilterer{}
	for _, filter := range f.cachedFilters {
		filters = append(filters, filter)
	}
	return filters
}
