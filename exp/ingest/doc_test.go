package ingest

import (
	"context"
	"fmt"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

// Example_ledgerentrieshistoryarchive demonstrates how to stream all ledger
// entries live at specific checkpoint ledger from history archives.
func Example_ledgerentrieshistoryarchive() {
	archiveURL := "http://history.stellar.org/prd/core-live/core_live_001"

	archive, err := historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{Context: context.TODO()},
	)
	if err != nil {
		panic(err)
	}

	// Ledger must be a checkpoint ledger: (100031+1) mod 64 == 0.
	reader, err := io.MakeSingleLedgerStateReader(context.TODO(), archive, 100031)
	if err != nil {
		panic(err)
	}

	var accounts, data, trustlines, offers int
	for {
		entry, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		switch entry.Type {
		case xdr.LedgerEntryTypeAccount:
			accounts++
		case xdr.LedgerEntryTypeData:
			data++
		case xdr.LedgerEntryTypeTrustline:
			trustlines++
		case xdr.LedgerEntryTypeOffer:
			offers++
		default:
			panic("Unknown type")
		}
	}

	fmt.Println("accounts", accounts)
	fmt.Println("data", data)
	fmt.Println("trustlines", trustlines)
	fmt.Println("offers", offers)
}

// Example_transactionshistoryarchive demonstrates how to stream transactions
// for a specific ledger from history archives. Please note that transaction
// meta IS NOT available in history archives.
func Example_transactionshistoryarchive() {
	archiveURL := "http://history.stellar.org/prd/core-live/core_live_001"
	networkPassphrase := network.PublicNetworkPassphrase

	archive, err := historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{Context: context.TODO()},
	)
	if err != nil {
		panic(err)
	}

	backend := ledgerbackend.NewHistoryArchiveBackendFromArchive(archive)
	txReader, err := io.NewLedgerTransactionReader(backend, networkPassphrase, 30000000)
	if err != nil {
		panic(err)
	}

	for {
		tx, err := txReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		fmt.Printf("%d: %x (%d ops)\n", tx.Index, tx.Result.TransactionHash, len(tx.Envelope.Operations()))
	}
}

// Example_changes demonstrates how to stream ledger entry changes
// for a specific ledger using captive stellar-core. Please note that transaction
// meta IS available when using this backend.
func Example_changes() {
	archiveURL := "http://history.stellar.org/prd/core-live/core_live_001"
	networkPassphrase := network.PublicNetworkPassphrase

	// Requires Stellar-Core 13.2.0+
	backend, err := ledgerbackend.NewCaptive(
		"/bin/stellar-core",
		"/opt/stellar-core.cfg",
		networkPassphrase,
		[]string{archiveURL},
	)
	if err != nil {
		panic(err)
	}

	sequence := uint32(3)

	err = backend.PrepareRange(ledgerbackend.SingleLedgerRange(sequence))
	if err != nil {
		panic(err)
	}

	changeReader, err := io.NewLedgerChangeReader(backend, networkPassphrase, sequence)
	if err != nil {
		panic(err)
	}

	for {
		change, err := changeReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		var action string
		switch {
		case change.Pre == nil && change.Post != nil:
			action = "created"
		case change.Pre != nil && change.Post != nil:
			action = "updated"
		case change.Pre != nil && change.Post == nil:
			action = "removed"
		}

		switch change.Type {
		case xdr.LedgerEntryTypeAccount:
			var accountEntry xdr.AccountEntry
			if change.Pre != nil {
				accountEntry = change.Pre.Data.MustAccount()
			} else {
				accountEntry = change.Post.Data.MustAccount()
			}
			fmt.Println("account", accountEntry.AccountId.Address(), action)
		case xdr.LedgerEntryTypeData:
			fmt.Println("data", action)
		case xdr.LedgerEntryTypeTrustline:
			fmt.Println("trustline", action)
		case xdr.LedgerEntryTypeOffer:
			fmt.Println("offer", action)
		default:
			panic("Unknown type")
		}
	}
}
