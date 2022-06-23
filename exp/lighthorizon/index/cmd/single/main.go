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
	start := flag.Int("start", 2, "ledger to start at (inclusive, default: earliest)")
	end := flag.Int("end", 0, "ledger to end at (inclusive, default: latest)")
	modules := flag.String("modules", "accounts,transactions", "comma-separated list of modules to index (default: all)")
	watch := flag.Bool("watch", false, "whether to watch the `source` for new "+
		"txmeta files and index them (default: false). "+
		"note: `-watch` implicitly implies `-end -1`")
	workerCount := flag.Int("workers", runtime.NumCPU()-1, "number of workers (default: # of CPUs - 1)")

	flag.Parse()
	log.SetLevel(log.InfoLevel)
	// log.SetLevel(log.DebugLevel)

	builder, err := index.BuildIndices(
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

	if *watch {
		if err := builder.Watch(context.Background()); err != nil {
			panic(err)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
