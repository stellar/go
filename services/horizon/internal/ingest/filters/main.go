package filters

import (
	"context"
	"time"

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
	assetFilter                    AssetFilter
	accountFilter                  AccountFilter
	lastFilterConfigCheckUnixEpoch int64
}

type Filters interface {
	GetFilters(filterQ history.QFilter, ctx context.Context) []processors.LedgerTransactionFilterer
}

func NewFilters() Filters {
	return &filtersCache{
		assetFilter:   NewAssetFilter(),
		accountFilter: NewAccountFilter(),
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

	if filterConfig, err := filterQ.GetAssetFilterConfig(ctx); err != nil {
		LOG.Errorf("unable to refresh asset filter config %v", err)
	} else {
		if err := f.assetFilter.RefreshAssetFilter(&filterConfig); err != nil {
			LOG.Errorf("unable to refresh asset filter config %v", err)
		}
	}

	if filterConfig, err := filterQ.GetAccountFilterConfig(ctx); err != nil {
		LOG.Errorf("unable to refresh account filter config %v", err)
	} else {
		if err := f.accountFilter.RefreshAccountFilter(&filterConfig); err != nil {
			LOG.Errorf("unable to refresh account filter config %v", err)
		}
	}

	return f.convertCacheToList()
}

func (f *filtersCache) convertCacheToList() []processors.LedgerTransactionFilterer {
	return []processors.LedgerTransactionFilterer{f.assetFilter, f.accountFilter}
}
