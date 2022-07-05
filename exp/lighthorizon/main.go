package main

import (
	"flag"
	"net/http"

	"github.com/go-chi/chi"
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

	indexStore, err := index.Connect(*indexesUrl)
	if err != nil {
		panic(err)
	}

	log.SetLevel(log.DebugLevel)
	log.Info("Starting lighthorizon!")

	ingestArchive, err := archive.NewIngestArchive(*sourceUrl, *networkPassphrase)
	if err != nil {
		panic(err)
	}
	defer ingestArchive.Close()

	archiveWrapper := archive.Wrapper{Archive: ingestArchive, Passphrase: *networkPassphrase}

	router := chi.NewMux()
    router.Method(http.MethodGet, "/accounts/{account_id}/transactions", actions.TxByAccount(archiveWrapper, indexStore))
	router.Method(http.MethodGet, "/accounts/{account_id}/operations", actions.OpsByAccount(archiveWrapper, indexStore))
	http.HandleFunc("/operations", actions.Operations(archiveWrapper, indexStore))
	http.HandleFunc("/transactions", actions.Transactions(archiveWrapper, indexStore))
	http.HandleFunc("/", actions.ApiDocs())

	log.Fatal(http.ListenAndServe(":8080", nil))
}
