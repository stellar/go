package main

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/exp/ingest/ledgerbackend"

	ingestadapters "github.com/stellar/go/exp/ingest/adapters"
)

const dbURI = "postgres://stellar:postgres@localhost:8002/core"

func main() {
	// useBackend()
	useAdapter()
}

func useAdapter() {
	lba := ingestadapters.LedgerBackendAdapter{}
	err := lba.Init("postgres", dbURI)
	if err != nil {
		log.Fatal(err)
	}

	ledgerSequence, err := lba.GetLatestLedgerSequence()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Latest ledger =", ledgerSequence)

	lrc, err := lba.GetLedger(ledgerSequence)
	if err != nil {
		log.Fatal(err)
	}

	s := lrc.GetSequence()
	fmt.Println("lrc sequence:", s)

	h := lrc.GetHeader()
	fmt.Println("lrc header:", h)

	for {
		lt, err := lrc.Read()
		if err == io.EOF {
			break
		}

		d := lt.Index
		fmt.Println("\nIndex:", d)

		a := lt.Result
		fmt.Println("TransactionResultPair:")
		fmt.Println("    fee charged", a.Result.FeeCharged)
		fmt.Println("    ext", a.Result.Ext)
		fmt.Println("    result", a.Result.Result)

		b := lt.Envelope
		fmt.Println("TransactionEnvelope:")
		fmt.Println("b", b.Tx)

		c := lt.Meta
		fmt.Println("TransactionMeta:")
		fmt.Println("    operations", c.Operations)
		fmt.Println("    V", c.V)
		fmt.Println("    V1.Operations", c.V1.Operations)
		fmt.Println("    V1.TxChanges", c.V1.TxChanges)
	}

	lrc.Close()
	lba.Close()
}

func useBackend() {
	// Initialise the database backend
	dbb := ledgerbackend.DatabaseBackend{}
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

	// fmt.Println(ledgerCloseMeta)

	fmt.Println("N transactions =", len(ledgerCloseMeta.TransactionEnvelope))
	fmt.Println("ledgerCloseMeta.Transaction:", ledgerCloseMeta.TransactionEnvelope)

	fmt.Println("N transactionReults =", len(ledgerCloseMeta.TransactionResult))
	fmt.Println("ledgerCloseMeta.TransactionResults:", ledgerCloseMeta.TransactionResult)

	fmt.Println("N transactionMeta =", len(ledgerCloseMeta.TransactionMeta))
	fmt.Println("ledgerCloseMeta.TransactionMeta:", ledgerCloseMeta.TransactionMeta)

	a := ledgerCloseMeta.TransactionResult[0]
	fmt.Println("TransactionResultPair:", a)
	fmt.Println("    fee charged", a.Result.FeeCharged)
	fmt.Println("    ext", a.Result.Ext)
	fmt.Println("    result", a.Result.Result)

	b := ledgerCloseMeta.TransactionEnvelope[0]
	fmt.Println("TransactionEnvelope:", b)
	fmt.Println("b", b.Tx)

	c := ledgerCloseMeta.TransactionMeta[0]
	fmt.Println("TransactionMeta", c.Operations)
	fmt.Println("    operations", c.Operations)
	fmt.Println("    V", c.V)
	fmt.Println("    V1.Operations", c.V1.Operations)
	fmt.Println("    V1.TxChanges", c.V1.TxChanges)
}
