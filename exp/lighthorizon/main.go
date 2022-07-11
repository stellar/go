package main

import (
	"flag"
	"net/http"

	"github.com/go-chi/chi"
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

	lightHorizon := services.LightHorizon{
		Transactions: services.TransactionsService{
			Archive:    ingestArchive,
			Passphrase: *networkPassphrase,
			IndexStore: indexStore,
		},
		Operations: services.OperationsService{
			Archive:    ingestArchive,
			Passphrase: *networkPassphrase,
			IndexStore: indexStore,
		},
	}

	router := chi.NewMux()
	router.Route("/accounts/{account_id}", func(r chi.Router) {
		r.MethodFunc(http.MethodGet, "/transactions", actions.NewTXByAccountHandler(lightHorizon))
		r.MethodFunc(http.MethodGet, "/operations", actions.NewOpsByAccountHandler(lightHorizon))
	})

	router.MethodFunc(http.MethodGet, "/operations", actions.Operations(lightHorizon))
	router.MethodFunc(http.MethodGet, "/transactions", actions.Transactions(lightHorizon))
	router.MethodFunc(http.MethodGet, "/", actions.ApiDocs())

	log.Fatal(http.ListenAndServe(":8080", router))
}
