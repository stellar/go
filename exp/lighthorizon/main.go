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
			Wrap: func(a historyarchive.ArchiveBackend) (historyarchive.ArchiveBackend, error) {
				// WARNING: No cache expiry yet! This will save checkpoints into a temp
				// dir, and only clear them when the program exits. So be careful you
				// don't call big queries, and accidentally fill your hard-drive!
				// Total checkpoints now are around 0.6TB.
				return historyarchive.MakeFsCacheBackend(a, "")
			},
		},
	)
	if err != nil {
		panic(err)
	}

	indexStore, err := index.NewS3Store(&aws.Config{Region: aws.String("us-east-1")}, 20)
	if err != nil {
		panic(err)
	}

	log.SetLevel(log.DebugLevel)
	log.Info("Starting lighthorizon!")

	archiveWrapper := archive.Wrapper{historyArchive}
	http.HandleFunc("/operations", actions.Operations(archiveWrapper, indexStore))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
