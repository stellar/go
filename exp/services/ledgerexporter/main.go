package main

import exporter "github.com/stellar/go/exp/services/ledgerexporter/internal"

func main() {
	app := exporter.NewApp()
	app.Run()
}
