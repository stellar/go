package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

func claimables() {
	// Open a history archive using our existing configuration details.
	historyArchive, err := historyarchive.Connect(
		config.HistoryArchiveURLs[0],
		historyarchive.ConnectOptions{
			NetworkPassphrase: config.NetworkPassphrase,
			S3Region:          "eu-west-1",
			UnsignedRequests:  false,
		},
	)
	panicIf(err)

	// First, we need to establish a safe fallback in case of any problems, so
	// we'll set a 10-second timeout.
	ctx, canceller := context.WithTimeout(context.Background(), 5*time.Second)
	defer canceller()

	// We pass 123455 because given a checkpoint frequency of 64 ledgers (the
	// default in `ConnectOptions`, above), 123455+1 mod 64 == 0. Incompatible
	// sequence numbers will likely result in 404 errors.
	reader, err := ingest.NewCheckpointChangeReader(ctx, historyArchive, 123455)
	panicIf(err)

	entries, newTotal, newCBs := 0, 0, 0
	for {
		entry, err := reader.Read()
		if err == io.EOF {
			break
		}
		panicIf(err)

		entries++
		isNewEntry := entry.LedgerEntryChangeType() == xdr.LedgerEntryChangeTypeLedgerEntryCreated
		if isNewEntry {
			newTotal++
		}

		switch entry.Type {
		case xdr.LedgerEntryTypeClaimableBalance:
			if isNewEntry {
				newCBs++
			}
		// these are included for completeness of the demonstration
		case xdr.LedgerEntryTypeAccount:
		case xdr.LedgerEntryTypeData:
		case xdr.LedgerEntryTypeTrustline:
		case xdr.LedgerEntryTypeOffer:
		default:
			panic(fmt.Errorf("Unknown type: %+v", entry.Type))
		}

		fmt.Printf("Processed %d ledger entry changes...\r", entries)
	}

	fmt.Println()
	fmt.Printf("%d/%d created entries were claimable balances\n", newCBs, newTotal)
}
