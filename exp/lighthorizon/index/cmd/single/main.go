package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
	"golang.org/x/sync/errgroup"
)

func main() {
	sourceUrl := flag.String("source", "gcs://horizon-archive-poc", "history archive url to read txmeta files")
	targetUrl := flag.String("target", "file://indexes", "where to write indexes")
	networkPassphrase := flag.String("network-passphrase", network.TestNetworkPassphrase, "network passphrase")
	start := flag.Int("start", -1, "ledger to start at (inclusive, default: earliest)")
	end := flag.Int("end", -1, "ledger to end at (inclusive, default: latest)")
	modules := flag.String("modules", "accounts,transactions", "comma-separated list of modules to index (default: all)")

	// Should we use runtime.NumCPU() for a reasonable default?
	// Yes, but leave a CPU open so I can actually use my PC while this runs.
	workerCount := flag.Int("workers", runtime.NumCPU()-1, "number of workers (default: # of CPUs - 1)")

	flag.Parse()
	log.SetLevel(log.InfoLevel)

	ctx := context.Background()

	indexStore, err := index.Connect(*targetUrl)
	if err != nil {
		panic(err)
	}

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

	startTime := time.Now()

	startLedger := uint32(max(*start, 2))
	endLedger := uint32(*end)
	if endLedger < 0 {
		latest, err := ledgerBackend.GetLatestLedgerSequence(ctx)
		if err != nil {
			panic(err)
		}
		endLedger = latest
	}
	ledgerCount := 1 + (endLedger - startLedger) // +1 because endLedger is inclusive
	parallel := max(1, *workerCount)

	log.Infof("Creating indices for ledger range: %d through %d (%d ledgers)",
		startLedger, endLedger, ledgerCount)
	log.Infof("Using %d workers", parallel)

	// Create a bunch of workers that process ledgers a checkpoint range at a
	// time (better than a ledger at a time to minimize flushes).
	wg, ctx := errgroup.WithContext(ctx)
	ch := make(chan historyarchive.Range, parallel)

	indexBuilder := index.NewIndexBuilder(indexStore, *ledgerBackend, *networkPassphrase)
	for _, part := range strings.Split(*modules, ",") {
		switch part {
		case "transactions":
			indexBuilder.RegisterModule(index.ProcessTransaction)
		case "accounts":
			indexBuilder.RegisterModule(index.ProcessAccounts)
		default:
			panic(fmt.Errorf("Unknown module: %s", part))
		}
	}

	// Submit the work to the channels, breaking up the range into checkpoints.
	go func() {
		// Recall: A ledger X is a checkpoint ledger iff (X + 1) % 64 == 0
		nextCheckpoint := (((startLedger / 64) * 64) + 63)

		ledger := startLedger
		nextLedger := ledger + (nextCheckpoint - startLedger)
		for ledger <= endLedger {
			ch <- historyarchive.Range{Low: ledger, High: nextLedger}

			ledger = nextLedger + 1
			// Ensure we don't exceed the upper ledger bound
			nextLedger = uint32(min(int(endLedger), int(ledger+63)))
		}

		close(ch)
	}()

	processed := uint64(0)
	for i := 0; i < parallel; i++ {
		wg.Go(func() error {
			for ledgerRange := range ch {
				count := (ledgerRange.High - ledgerRange.Low) + 1
				nprocessed := atomic.AddUint64(&processed, uint64(count))

				log.Debugf("Working on checkpoint range %+v", ledgerRange)

				// Assertion for testing
				if ledgerRange.High != endLedger &&
					(ledgerRange.High+1)%64 != 0 {
					log.Fatalf("Uh oh: bad range")
				}

				err = indexBuilder.Build(ctx, ledgerRange)
				if err != nil {
					return err
				}

				printProgress("Reading ledgers",
					nprocessed, uint64(ledgerCount), startTime)

				// Upload indices once per checkpoint to save memory
				if err := indexStore.Flush(); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		panic(err)
	}

	printProgress("Reading ledgers",
		uint64(ledgerCount), uint64(ledgerCount), startTime)

	// Assertion for testing
	if processed != uint64(ledgerCount) {
		log.Fatalf("processed %d but expected %d", processed, ledgerCount)
	}

	log.Infof("Processed %d ledgers via %d workers", processed, parallel)
	log.Infof("Uploading indices to %s", *targetUrl)
	if err := indexStore.Flush(); err != nil {
		panic(err)
	}
}

func printProgress(prefix string, done, total uint64, startTime time.Time) {
	// This should never happen, more of a runtime assertion for now.
	// We can remove it when production-ready.
	if done > total {
		panic(fmt.Errorf("error for %s: done > total (%d > %d)",
			prefix, done, total))
	}

	progress := float64(done) / float64(total)
	elapsed := time.Since(startTime)

	// Approximate based on how many ledgers are left and how long this much
	// progress took, e.g. if 4/10 took 2s then 6/10 will "take" 3s (though this
	// assumes consistent ledger load).
	remaining := (float64(elapsed) / float64(done)) * float64(total-done)

	var remainingStr string
	if math.IsInf(remaining, 0) || math.IsNaN(remaining) {
		remainingStr = "unknown"
	} else {
		remainingStr = time.Duration(remaining).Round(time.Millisecond).String()
	}

	log.Infof("%s - %.1f%% (%d/%d) - elapsed: %s, remaining: ~%s", prefix,
		100*progress, done, total,
		elapsed.Round(time.Millisecond),
		remainingStr,
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
