package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/exp/ingest"
)

const dbURI = "postgres://stellar:postgres@localhost:8002/core"

func main() {
	// Initialise the database backend
	dbb := ingest.DatabaseBackend{}
	err := dbb.CreateSession("postgres", dbURI)
	if err != nil {
		log.Fatalf("Couldn't connect to database at %s: %s", dbURI, err)
	}
	defer dbb.Close()

	ledgerSequence, err := dbb.GetLatestLedgerSequence()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Latest ledger =", ledgerSequence)

	exists, ledgerCloseMeta, err := dbb.GetLedger(ledgerSequence)

	if err != nil {
		log.Fatal("error reading ledger from backend: ", err)
	}
	if !exists {
		log.Fatalf("Ledger %d was not found", ledgerSequence)
	}

	fmt.Println(ledgerCloseMeta)

	fmt.Println("N transactions =", len(ledgerCloseMeta.TransactionEnvelope))
	fmt.Println("ledgerCloseMeta.Transaction:", ledgerCloseMeta.TransactionEnvelope)

	fmt.Println("N transactionReults =", len(ledgerCloseMeta.TransactionResult))
	fmt.Println("ledgerCloseMeta.TransactionResults:", ledgerCloseMeta.TransactionResult)

	fmt.Println("N transactionMeta =", len(ledgerCloseMeta.TransactionMeta))
	fmt.Println("ledgerCloseMeta.TransactionMeta:", ledgerCloseMeta.TransactionMeta)
}
