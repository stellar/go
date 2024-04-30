package main

import (
	"flag"

	exporter "github.com/stellar/go/exp/services/ledgerexporter/internal"
)

func main() {
	flags := exporter.Flags{}
	startLedger := uint(0)
	endLedger := uint(0)
	flag.UintVar(&startLedger, "start", 0, "Starting ledger")
	flag.UintVar(&endLedger, "end", 0, "Ending ledger (inclusive)")
	flag.StringVar(&flags.ConfigFilePath, "config-file", "config.toml", "Path to the TOML config file")
	flag.BoolVar(&flags.Resume, "resume", false, "Attempt to find a resumable starting point on remote data store")
	flag.UintVar(&flags.AdminPort, "admin-port", 0, "Admin HTTP port for prometheus metrics")

	flag.Parse()
	flags.StartLedger = uint32(startLedger)
	flags.EndLedger = uint32(endLedger)

	app := exporter.NewApp(flags)
	app.Run()
}
