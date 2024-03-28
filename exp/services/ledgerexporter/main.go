package main

import (
	"flag"

	exporter "github.com/stellar/go/exp/services/ledgerexporter/internal"
)

func main() {
	flags := exporter.Flags{}
	flag.UintVar(&flags.StartLedger, "start", 0, "Starting ledger")
	flag.UintVar(&flags.EndLedger, "end", 0, "Ending ledger (inclusive)")
	flag.StringVar(&flags.ConfigFilePath, "config-file", "config.toml", "Path to the TOML config file")
	flag.Parse()

	app := exporter.NewApp(flags)
	app.Run()
}
