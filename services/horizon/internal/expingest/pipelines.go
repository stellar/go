package expingest

import (
	"context"
	"fmt"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/processors"
	"github.com/stellar/go/exp/orderbook"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	horizonProcessors "github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	ilog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

func accountForSignerStateNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.StateNode(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}).
		Pipe(
			pipeline.StateNode(&horizonProcessors.DatabaseProcessor{
				HistoryQ: q,
				Action:   horizonProcessors.AccountsForSigner,
			}),
		)
}

func orderBookDBStateNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.StateNode(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeOffer}).
		Pipe(
			pipeline.StateNode(&horizonProcessors.DatabaseProcessor{
				OffersQ: q,
				Action:  horizonProcessors.Offers,
			}),
		)
}

func orderBookGraphStateNode(graph *orderbook.OrderBookGraph) *supportPipeline.PipelineNode {
	return pipeline.StateNode(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeOffer}).
		Pipe(
			pipeline.StateNode(&horizonProcessors.OrderbookProcessor{
				OrderBookGraph: graph,
			}),
		)
}

func buildStatePipeline(children []*supportPipeline.PipelineNode) *pipeline.StatePipeline {
	statePipeline := &pipeline.StatePipeline{}

	statePipeline.SetRoot(
		pipeline.StateNode(&processors.RootProcessor{}).
			Pipe(children...),
	)

	return statePipeline
}

func accountForSignerLedgerNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.LedgerNode(&horizonProcessors.DatabaseProcessor{
		HistoryQ: q,
		Action:   horizonProcessors.AccountsForSigner,
	})
}

func orderBookDBLedgerNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.LedgerNode(&horizonProcessors.DatabaseProcessor{
		OffersQ: q,
		Action:  horizonProcessors.Offers,
	})
}

func orderBookGraphLedgerNode(graph *orderbook.OrderBookGraph) *supportPipeline.PipelineNode {
	return pipeline.LedgerNode(&horizonProcessors.OrderbookProcessor{
		OrderBookGraph: graph,
	})
}

func buildLedgerPipeline(children []*supportPipeline.PipelineNode) *pipeline.LedgerPipeline {
	ledgerPipeline := &pipeline.LedgerPipeline{}

	ledgerPipeline.SetRoot(
		pipeline.LedgerNode(&processors.RootProcessor{}).Pipe(children...),
	)

	return ledgerPipeline
}

func addPipelineHooks(
	p supportPipeline.PipelineInterface,
	historySession *db.Session,
	ingestSession ingest.Session,
	graph *orderbook.OrderBookGraph,
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
		defer graph.Discard()

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

		if err := historyQ.UpdateLastLedgerExpIngest(ledgerSeq); err != nil {
			return errors.Wrap(err, "Error updating last ingested ledger")
		}

		if err := historyQ.UpdateExpIngestVersion(CurrentVersion); err != nil {
			return errors.Wrap(err, "Error updating expingest version")
		}

		if err := historySession.Commit(); err != nil {
			return errors.Wrap(err, "Error commiting db transaction")
		}

		if graph != nil {
			if err := graph.Apply(); err != nil {
				return errors.Wrap(err, "Error applying order book changes")
			}
		}

		log.WithFields(ilog.F{"ledger": ledgerSeq, "type": pipelineType}).Info("Processed ledger")
		return nil
	})
}
