package main

import (
	"fmt"

	"github.com/stellar/go/ingest/ledgerbackend"
)

var (
	config = ledgerbackend.CaptiveCoreConfig{
		// Change these based on your environment:
		BinaryPath:        "/usr/local/bin/stellar-core",
		ConfigAppendPath:  "stellar-core-stub.toml",
		NetworkPassphrase: "Test SDF Network ; September 2015",
		HistoryArchiveURLs: []string{
			"https://history.stellar.org/prd/core-testnet/core_testnet_001",
			"https://history.stellar.org/prd/core-testnet/core_testnet_002",
			"https://history.stellar.org/prd/core-testnet/core_testnet_003",
		},
	}
)

func panicIf(err error) {
	if err != nil {
		panic(fmt.Errorf("An error occurred, panicking: %s\n", err))
	}
}
