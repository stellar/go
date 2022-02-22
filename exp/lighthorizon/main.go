package main

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stellar/go/exp/lighthorizon/actions"
	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
)

func main() {
	historyArchive, err := historyarchive.Connect(
		"http://s3-eu-west-1.amazonaws.com/history.stellar.org/prd/core-live/core_live_001",
		historyarchive.ConnectOptions{
			NetworkPassphrase: network.PublicNetworkPassphrase,
			S3Region:          "us-west-1",
			UnsignedRequests:  false,
		},
	)
	if err != nil {
		panic(err)
	}

	indexStore, err := index.NewS3IndexStore(&aws.Config{Region: aws.String("us-east-1")}, 20)
	if err != nil {
		panic(err)
	}

	log.SetLevel(log.DebugLevel)
	log.Info("Starting lighthorizon!")

	archiveWrapper := archive.Wrapper{historyArchive}
	http.HandleFunc("/operations", actions.Operations(archiveWrapper, indexStore))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
