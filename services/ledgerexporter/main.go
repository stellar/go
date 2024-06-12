package main

import (
	"fmt"
	"os"

	exporter "github.com/stellar/go/services/ledgerexporter/internal"
)

func main() {
	err := exporter.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
