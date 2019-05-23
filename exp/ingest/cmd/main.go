package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/exp/ingest"
)

const dbURI = "postgres://stellar:postgres@localhost:8002/core"

const txHistoryQuery = "select * from txhistory limit 10;"

func main() {
	coreSession, err := ingest.CreateSession("postgres", dbURI)
	defer coreSession.DB.Close()

	if err != nil {
		log.Fatalf("Couldn't connect to database at %s: %s", dbURI, err)
	}

	var rows []TXHistory
	err = coreSession.SelectRaw(&rows, txHistoryQuery)
	if err != nil {
		log.Fatal("Couldn't select txhistory rows: ", err)
	}

	for _, row := range rows {
		fmt.Println(row.TXID)
	}
}

type TXHistory struct {
	TXID      string `db:"txid"`
	LedgerSeq int    `db:"ledgerseq"`
	TXIndex   int    `db:"txindex"`
	TXBody    string `db:"txbody"`
	TXResult  string `db:"txresult"`
	TXMeta    string `db:"txmeta"`
}
