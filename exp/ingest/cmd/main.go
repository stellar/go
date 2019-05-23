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

	rows, err := dbb.GetTXHistory()
	if err != nil {
		log.Fatal("Couldn't select txhistory rows: ", err)
	}

	for _, row := range rows {
		fmt.Println(row.TXID)
	}

	ledgerSequence, err := dbb.GetLatestLedgerSequence()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Latest ledger =", ledgerSequence)
}
