package main

import (
	"flag"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"

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

	archiveWrapper := archive.Wrapper{Archive: ingestArchive, Passphrase: *networkPassphrase}

	http.HandleFunc("/", actions.ApiDocs())
	http.HandleFunc("/operations", actions.Operations(archiveWrapper, indexStore))
	http.HandleFunc("/transactions", actions.Transactions(archiveWrapper, indexStore))

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
