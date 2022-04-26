package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
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

	// Simple file os access
	source, err := historyarchive.ConnectBackend(
		*sourceUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: *networkPassphrase,
		},
	)
	if err != nil {
		panic(err)
	}
	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(source)
	defer ledgerBackend.Close()
	archiveWrapper := archive.Wrapper{Archive: ledgerBackend, Passphrase: *networkPassphrase}
	http.HandleFunc("/operations", actions.Operations(archiveWrapper, indexStore))
	http.HandleFunc("/transactions", actions.Transactions(archiveWrapper, indexStore))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
