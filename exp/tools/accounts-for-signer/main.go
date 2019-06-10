package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/processors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

func main() {
	archive, err := archive()
	if err != nil {
		panic(err)
	}

	session := &ingest.SingleLedgerSession{Archive: archive}
	p := pipeline.New(
		// Passes accounts only
		pipeline.Node(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}).
			Pipe(
				// Finds accounts for a single signer
				pipeline.Node(&AccountsForSignerProcessor{Signer: "GBMALBYJT6A73SYQWOWVVCGSPUPJPBX4AFDJ7A63GG64QCNRCAFYWWEN"}).
					Pipe(pipeline.Node(&processors.CSVPrinter{Filename: "./accounts_for_signer.csv"})),
			),
	)

	doneStats := printPipelineStats(p)

	session.AddPipeline(p)

	err = session.Run()
	if err != nil {
		fmt.Println("Session errored:")
		fmt.Println(err)
	} else {
		fmt.Println("Session finished without errors")
	}

	time.Sleep(10 * time.Second)
	doneStats <- true
	time.Sleep(10 * time.Second)
	// Print go routines count for the last time
	fmt.Printf("Goroutines = %v\n", runtime.NumGoroutine())
}

func archive() (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		fmt.Sprintf("s3://history.stellar.org/prd/core-live/core_live_001/"),
		historyarchive.ConnectOptions{
			S3Region:         "eu-west-1",
			UnsignedRequests: true,
		},
	)
}

func buildPipeline() (*pipeline.StatePipeline, error) {
	p := &pipeline.StatePipeline{}

	p.SetRoot(
		// Passes accounts only
		p.Node(&EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}).
			Pipe(
				// Finds accounts for a single signer
				p.Node(&AccountsForSignerProcessor{Signer: "GBMALBYJT6A73SYQWOWVVCGSPUPJPBX4AFDJ7A63GG64QCNRCAFYWWEN"}).
					Pipe(p.Node(&PrintAllProcessor{Filename: "./accounts_for_signer.csv"})),
			),
	)

	return p, nil
}

func printPipelineStats(p *pipeline.Pipeline) chan<- bool {
	startTime := time.Now()
	done := make(chan bool)
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()

		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
			fmt.Printf("\tHeapAlloc = %v MiB", bToMb(m.HeapAlloc))
			fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
			fmt.Printf("\tNumGC = %v", m.NumGC)
			fmt.Printf("\tGoroutines = %v", runtime.NumGoroutine())
			fmt.Printf("\tNumCPU = %v\n\n", runtime.NumCPU())

			fmt.Printf("Duration: %s\n", time.Since(startTime))
			fmt.Println("Pipeline status:")
			p.PrintStatus()

			fmt.Println("========================================")

			select {
			case <-ticker.C:
				continue
			case <-done:
				// Pipeline done
				return
			}
		}
	}()

	return done
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
