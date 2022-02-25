package filters

import (
	"context"
	"sync"
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
	supportedFilterNames           = []string{FilterAssetFilterName, FilterAccountFilterName}
	loadedFilters                  []processors.LedgerTransactionFilterer
	lastFilterConfigCheckUnixEpoch int64
	LOG                            = log.WithFields(log.F{
		"filters": "load",
	})
	lock sync.Mutex
)

// Provide list of the active filters. Optimize performance by caching the list, only
// rebuild the list on expiration time interval. Method is thread-safe.
func GetFilters(filterQ history.QFilter, ctx context.Context) []processors.LedgerTransactionFilterer {
	lock.Lock()
	defer lock.Unlock()
	// only attempt to refresh filter config cache state at configured interval limit
	if time.Now().Unix() < (lastFilterConfigCheckUnixEpoch + filterConfigCheckIntervalSeconds) {
		return append([]processors.LedgerTransactionFilterer{}, loadedFilters...)
	}

	loadedFilters = []processors.LedgerTransactionFilterer{}
	LOG.Info("expired filter config cache, refresh from db")
	filterConfigs, err := filterQ.GetAllFilters(ctx)
	if err != nil {
		LOG.Errorf("unable to query filter configs, %v", err)
		// reset the cache time regardless, so next attempt is at next interval
		lastFilterConfigCheckUnixEpoch = time.Now().Unix()
		return append([]processors.LedgerTransactionFilterer{}, loadedFilters...)
	}

	loadedFilters := []processors.LedgerTransactionFilterer{}
	for _, filterConfig := range filterConfigs {
		if filterConfig.Enabled {
			switch filterConfig.Name {
			case FilterAssetFilterName:
				assetFilter, err := GetAssetFilter(&filterConfig)
				if err != nil {
					LOG.Errorf("unable to create asset filter %v", err)
					continue
				}
				loadedFilters = append(loadedFilters, assetFilter)
			case FilterAccountFilterName:
				accountFilter, err := GetAccountFilter(&filterConfig)
				if err != nil {
					LOG.Errorf("unable to create asset filter %v", err)
					continue
				}
				loadedFilters = append(loadedFilters, accountFilter)
			}

		}
	}
	return append([]processors.LedgerTransactionFilterer{}, loadedFilters...)
}

func SupportedFilterNames(name string) bool {
	for _, supportedName := range supportedFilterNames {
		if name == supportedName {
			return true
		}
	}
	return false
}
