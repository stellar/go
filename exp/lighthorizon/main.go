package main

import (
	"flag"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/exp/lighthorizon/services"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
)

func main() {
	sourceUrl := flag.String("source", "gcs://horizon-archive-poc", "history archive url to read txmeta files")
	indexesUrl := flag.String("indexes", "file://indexes", "url of the indexes")
	networkPassphrase := flag.String("network-passphrase", network.TestNetworkPassphrase, "network passphrase")
	flag.Parse()

	L := log.WithField("service", "horizon-lite")
	// L.SetLevel(log.DebugLevel)
	L.Info("Starting lighthorizon!")

	registry := prometheus.NewRegistry()
	indexStore, err := index.ConnectWithConfig(index.StoreConfig{
		Url:     *indexesUrl,
		Metrics: registry,
		Log:     L.WithField("subservice", "index"),
	})
	if err != nil {
		panic(err)
	}

	ingestArchive, err := archive.NewIngestArchive(*sourceUrl, *networkPassphrase)
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

	router := chi.NewMux()
	router.Route("/accounts/{account_id}", func(r chi.Router) {
		r.MethodFunc(http.MethodGet, "/transactions", actions.NewTXByAccountHandler(lightHorizon))
		r.MethodFunc(http.MethodGet, "/operations", actions.NewOpsByAccountHandler(lightHorizon))
	})

	router.MethodFunc(http.MethodGet, "/", actions.ApiDocs())
	router.Method(http.MethodGet, "/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Fatal(http.ListenAndServe(":8080", router))
}
