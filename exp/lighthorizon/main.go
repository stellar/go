package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
)

func main() {
	targetUrl := flag.String("target", "gcs://horizon-archive-poc", "history archive url to read txmeta files")
	networkPassphrase := flag.String("network-passphrase", network.TestNetworkPassphrase, "network passphrase")
	flag.Parse()

	indexStore, err := index.NewS3Store(&aws.Config{Region: aws.String("us-east-1")}, "", 20)
	if err != nil {
		panic(err)
	}

	log.SetLevel(log.DebugLevel)
	log.Info("Starting lighthorizon!")

	// Simple file os access
	target, err := historyarchive.ConnectBackend(
		*targetUrl,
		historyarchive.ConnectOptions{
			Context:           context.Background(),
			NetworkPassphrase: *networkPassphrase,
		},
	)
	if err != nil {
		panic(err)
	}
	ledgerBackend := ledgerbackend.NewHistoryArchiveBackend(target)
	defer ledgerBackend.Close()
	archiveWrapper := archive.Wrapper{Archive: ledgerBackend, Passphrase: *networkPassphrase}
	http.HandleFunc("/operations", actions.Operations(archiveWrapper, indexStore))
	http.HandleFunc("/transactions", actions.Transactions(archiveWrapper, indexStore))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
