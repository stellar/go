package main

import (
	"flag"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/exp/lighthorizon/services"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
)

const (
	defaultCacheSize = (60 * 60 * 24) / 6 // 1 day of ledgers @ 6s each
)

func main() {
	sourceUrl := flag.String("source", "gcs://horizon-archive-poc", "history archive url to read txmeta files")
	indexesUrl := flag.String("indexes", "file://indexes", "url of the indexes")
	networkPassphrase := flag.String("network-passphrase", network.PublicNetworkPassphrase, "network passphrase")
	cacheDir := flag.String("ledger-cache", "", `path to cache frequently-used ledgers;
if left empty, uses a temporary directory`)
	cacheSize := flag.Int("ledger-cache-size", defaultCacheSize,
		"number of ledgers to store in the cache")
	flag.Parse()

	L := log.WithField("service", "horizon-lite")
	L.SetLevel(log.InfoLevel)
	L.Info("Starting lighthorizon!")

	registry := prometheus.NewRegistry()
	indexStore, err := index.ConnectWithConfig(index.StoreConfig{
		Url:     *indexesUrl,
		Log:     L.WithField("subservice", "index"),
		Metrics: registry,
	})
	if err != nil {
		panic(err)
	}

	ingestArchive, err := archive.NewIngestArchive(archive.ArchiveConfig{
		SourceUrl:         *sourceUrl,
		NetworkPassphrase: *networkPassphrase,
		CacheDir:          *cacheDir,
		CacheSize:         *cacheSize,
	})
	if err != nil {
		panic(err)
	}
	defer ingestArchive.Close()

	Config := services.Config{
		Archive:    ingestArchive,
		Passphrase: *networkPassphrase,
		IndexStore: indexStore,
	}

	lightHorizon := services.LightHorizon{
		Transactions: services.TransactionsService{
			Config: Config,
		},
		Operations: services.OperationsService{
			Config: Config,
		},
	}

	log.Fatal(http.ListenAndServe(":8080", lightHorizonHTTPHandler(registry, lightHorizon)))
}
