package main

import (
	"fmt"
	"time"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/processors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

func main() {
	session := &ingest.LiveSession{
		Archive:       archive(),
		LedgerBackend: ledgerBackend(),
	}

	session.SetStatePipeline(buildStatePipeline())
	session.SetLedgerPipeline(buildLedgerPipeline())

	go func() {
		time.Sleep(time.Minute)
		fmt.Println("Shutting down. Remove next line to run full demo.")
		session.Shutdown()
	}()

	err := session.Run()
	if err != nil {
		panic(err)
	}
}

func archive() *historyarchive.Archive {
	a, err := historyarchive.Connect(
		fmt.Sprintf("s3://history.stellar.org/prd/core-live/core_live_001/"),
		historyarchive.ConnectOptions{
			S3Region:         "eu-west-1",
			UnsignedRequests: true,
		},
	)
	if err != nil {
		panic(err)
	}
	return a
}

func ledgerBackend() ledgerbackend.LedgerBackend {
	return &ledgerbackend.DatabaseBackend{
		DataSourceName: "postgres://localhost:5432/core?sslmode=disable",
	}
}

func buildStatePipeline() *pipeline.StatePipeline {
	statePipeline := &pipeline.StatePipeline{}

	statePipeline.SetRoot(
		// Prints number of read entries every N entries...
		statePipeline.Node(&processors.StatusLogger{N: 5000}).
			Pipe(
				// Passes accounts only
				statePipeline.Node(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}).
					Pipe(
						statePipeline.Node(&processors.CSVPrinter{Filename: "accounts.csv"}),
					),
			),
	)

	return statePipeline
}

func buildLedgerPipeline() *pipeline.LedgerPipeline {
	ledgerPipeline := &pipeline.LedgerPipeline{}

	ledgerPipeline.SetRoot(
		ledgerPipeline.Node(&processors.CSVPrinter{}),
	)

	return ledgerPipeline
}
