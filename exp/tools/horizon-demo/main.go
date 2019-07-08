package main

import (
	"fmt"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
)

func main() {
	db, err := NewDatabase("postgres://localhost:5432/horizondemo?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	orderBookGraph := orderbook.NewOrderBookGraph()

	session := &ingest.LiveSession{
		Archive:       archive(),
		LedgerBackend: ledgerBackend(),

		StatePipeline:  buildStatePipeline(db, orderBookGraph),
		LedgerPipeline: buildLedgerPipeline(db, orderBookGraph),
	}

	addPipelineHooks(session.StatePipeline, db, session, orderBookGraph)
	addPipelineHooks(session.LedgerPipeline, db, session, orderBookGraph)

	printPipelinesStats(session.StatePipeline, session.LedgerPipeline)

	// This is broken when the last ledger does not contain transactions
	// but it's just a demo (we don't store ledgers, just transactions).
	ledger, err := db.GetLatestLedger()
	if err != nil && !db.NoRows(errors.Cause(err)) {
		panic(err)
	}

	if ledger == 0 {
		err = session.Run()
	} else {
		err = session.Resume(ledger + 1)
	}

	if err != nil {
		panic(err)
	}
}

func archive() *historyarchive.Archive {
	a, err := historyarchive.Connect(
		fmt.Sprintf("s3://history.stellar.org/prd/core-live/core_live_001/"),
		historyarchive.ConnectOptions{
			S3Region:         "eu-west-1",
			UnsignedRequests: true,
		},
	)
	if err != nil {
		panic(err)
	}
	return a
}
