package main

import (
	"context"
	"fmt"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/processors"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

func main() {
	db, err := NewDatabase("postgres://localhost:5432/horizondemo?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	session := &ingest.LiveSession{
		Archive:       archive(),
		LedgerBackend: ledgerBackend(),

		StatePipeline:  buildStatePipeline(db),
		LedgerPipeline: buildLedgerPipeline(db),
	}

	addPipelineHooks(session.StatePipeline, db, session)
	addPipelineHooks(session.LedgerPipeline, db, session)

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

func ledgerBackend() ledgerbackend.LedgerBackend {
	return &ledgerbackend.DatabaseBackend{
		DataSourceName: "postgres://localhost:5432/core?sslmode=disable",
	}
}

func buildStatePipeline(db *Database) *pipeline.StatePipeline {
	statePipeline := &pipeline.StatePipeline{}

	statePipeline.SetRoot(
		// Prints number of read entries every N entries...
		statePipeline.Node(&processors.StatusLogger{N: 5000}).
			Pipe(
				statePipeline.Node(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}).
					Pipe(
						statePipeline.Node(&DatabaseProcessor{
							Database: db,
							Action:   AccountsForSigner,
						}),
					),
			),
	)

	return statePipeline
}

func buildLedgerPipeline(db *Database) *pipeline.LedgerPipeline {
	ledgerPipeline := &pipeline.LedgerPipeline{}

	ledgerPipeline.SetRoot(
		ledgerPipeline.Node(&processors.RootProcessor{}).Pipe(
			ledgerPipeline.Node(&DatabaseProcessor{
				Database: db,
				Action:   AccountsForSigner,
			}),
			ledgerPipeline.Node(&DatabaseProcessor{
				Database: db,
				Action:   Transactions,
			}),
		),
	)

	return ledgerPipeline
}

func addPipelineHooks(p supportPipeline.PipelineInterface, db *Database, session ingest.Session) {
	p.AddPreProcessingHook(func(ctx context.Context) error {
		ledgerSeq := pipeline.GetLedgerSequenceFromContext(ctx)
		fmt.Printf("Processing ledger: %d\n", ledgerSeq)
		return db.Begin()
	})

	p.AddPostProcessingHook(func(ctx context.Context, err error) error {
		ledgerSeq := pipeline.GetLedgerSequenceFromContext(ctx)

		if err != nil {
			fmt.Println("Error processing ledger:", err)
			return db.Rollback()
		}

		fmt.Printf("Processed ledger: %d\n", ledgerSeq)

		// Acquire write lock
		session.UpdateLock()
		defer session.UpdateUnlock()

		return db.Commit()
	})
}
