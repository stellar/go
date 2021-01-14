package main

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/ingest"
	backends "github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/log"
)

var (
	config = backends.CaptiveCoreConfig{
		// Change these based on your env:
		BinaryPath:        "/usr/local/bin/stellar-core",
		ConfigAppendPath:  "stellar-core-stub.toml",
		NetworkPassphrase: "Test SDF Network ; September 2015",
		HistoryArchiveURLs: []string{
			"https://history.stellar.org/prd/core-testnet/core_testnet_001",
		},
	}
)

func main() {
	lg := log.New()
	lg.SetLevel(logrus.InfoLevel)
	lg.Entry.Logger.Out = ioutil.Discard
	config.Log = lg

	backend, err := backends.NewCaptive(config)
	panicIf(err)
	defer backend.Close()

	// Prepare a range to be ingested:
	var startingSeq uint32 = 2 // can't start with genesis ledger
	var ledgersToRead uint32 = 10000

	fmt.Printf("Preparing range (%d ledgers)... ", ledgersToRead)
	ledgerRange := backends.BoundedRange(startingSeq, startingSeq+ledgersToRead)
	err = backend.PrepareRange(ledgerRange)
	panicIf(err)
	fmt.Println("done.")

	// These are the statistics that we're tracking.
	var successfulTransactions, failedTransactions int
	var operationsInSuccessful, operationsInFailed int

	var delta uint32 = 0
	for ; delta <= ledgersToRead; delta++ {
		fmt.Printf("Processed %d/%d ledgers (%0.1f%%)... ",
			delta, ledgersToRead, 100*float32(delta)/float32(ledgersToRead))

		txReader, err := ingest.NewLedgerTransactionReader(
			backend, config.NetworkPassphrase, startingSeq+delta)
		panicIf(err)
		defer txReader.Close()

		for {
			tx, err := txReader.Read()
			if err == io.EOF {
				break
			}
			panicIf(err)

			envelope := tx.Envelope
			operationCount := len(envelope.Operations())
			if tx.Result.Successful() {
				successfulTransactions++
				operationsInSuccessful += operationCount
			} else {
				failedTransactions++
				operationsInFailed += operationCount
			}
		}

		if delta < ledgersToRead {
			fmt.Printf("\r")
		}
	}

	fmt.Println("done.")
	fmt.Println("Results:")
	fmt.Printf("  - total transactions: %d\n", successfulTransactions+failedTransactions)
	fmt.Printf("  - succeeded / failed: %d / %d\n", successfulTransactions, failedTransactions)
	fmt.Printf("  - total operations:   %d\n", operationsInSuccessful+operationsInFailed)
	fmt.Printf("  - succeeded / failed: %d / %d\n", operationsInSuccessful, operationsInFailed)
}

func panicIf(err error) {
	if err != nil {
		panic(fmt.Errorf("An error occurred, panicking: %s\n", err))
	}
}
