package main

import (
	"net/http"

	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
)

func main() {
	historyArchive, err := historyarchive.Connect(
		"https://history.stellar.org/prd/core-live/core_live_002",
		historyarchive.ConnectOptions{
			NetworkPassphrase: network.PublicNetworkPassphrase,
			S3Region:          "us-west-1",
			UnsignedRequests:  false,
		},
	)
	if err != nil {
		panic(err)
	}

	log.SetLevel(log.DebugLevel)
	log.Info("Starting lighthorizon!")

	archiveWrapper := archive.Wrapper{historyArchive}
	http.HandleFunc("/operations", actions.Operations(archiveWrapper))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
