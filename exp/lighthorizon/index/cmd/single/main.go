package main

import (
	"context"
	"flag"
	"runtime"
	"strings"

	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
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

	err := index.BuildIndices(
		context.Background(),
		*sourceUrl,
		*targetUrl,
		*networkPassphrase,
		uint32(max(*start, 2)),
		uint32(*end),
		strings.Split(*modules, ","),
		*workerCount,
	)
	if err != nil {
		panic(err)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
