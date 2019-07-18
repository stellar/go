package expingest

import (
	"context"
	"fmt"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/processors"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	horizonProcessors "github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	ilog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

func buildStatePipeline(q *history.Q) *pipeline.StatePipeline {
	statePipeline := &pipeline.StatePipeline{}

	statePipeline.SetRoot(
		statePipeline.Node(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}).
			Pipe(
				statePipeline.Node(&horizonProcessors.DatabaseProcessor{
					HistoryQ: q,
					Action:   horizonProcessors.AccountsForSigner,
				}),
			),
	)

	return statePipeline
}

func buildLedgerPipeline(q *history.Q) *pipeline.LedgerPipeline {
	ledgerPipeline := &pipeline.LedgerPipeline{}

	ledgerPipeline.SetRoot(
		ledgerPipeline.Node(&processors.RootProcessor{}).Pipe(
			ledgerPipeline.Node(&horizonProcessors.DatabaseProcessor{
				HistoryQ: q,
				Action:   horizonProcessors.AccountsForSigner,
			}),
		),
	)

	return ledgerPipeline
}

func addPipelineHooks(
	p supportPipeline.PipelineInterface,
	historySession *db.Session,
	ingestSession ingest.Session,
) {
	var pipelineType string
	switch p.(type) {
	case *pipeline.StatePipeline:
		pipelineType = "state_pipeline"
	case *pipeline.LedgerPipeline:
		pipelineType = "ledger_pipeline"
	default:
		panic(fmt.Sprintf("Unknown pipeline type %T", p))
	}

	historyQ := &history.Q{historySession}

	p.AddPreProcessingHook(func(ctx context.Context) error {
		ledgerSeq := pipeline.GetLedgerSequenceFromContext(ctx)
		log.WithFields(ilog.F{"ledger": ledgerSeq, "type": pipelineType}).Info("Processing ledger")
		return historySession.Begin()
	})

	p.AddPostProcessingHook(func(ctx context.Context, err error) error {
		defer historySession.Rollback()

		ledgerSeq := pipeline.GetLedgerSequenceFromContext(ctx)

		if err != nil {
			log.
				WithFields(ilog.F{
					"ledger": ledgerSeq,
					"type":   pipelineType,
					"err":    err,
				}).
				Error("Error processing ledger")
			return err
		}

		err = historyQ.UpdateLastLedgerExpIngest(ledgerSeq)
		if err != nil {
			return errors.Wrap(err, "Error updating last ingested ledger")
		}

		err = historySession.Commit()
		if err != nil {
			return errors.Wrap(err, "Error commiting db transaction")
		}

		log.WithFields(ilog.F{"ledger": ledgerSeq, "type": pipelineType}).Info("Processed ledger")
		return nil
	})
}
