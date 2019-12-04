package expingest

import (
	"context"
	"fmt"

	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/ingest/processors"
	"github.com/stellar/go/exp/ingest/verify"
	"github.com/stellar/go/exp/orderbook"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	horizonProcessors "github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

type pType string

const (
	statePipeline                   pType = "state_pipeline"
	ledgerPipeline                  pType = "ledger_pipeline"
	stateVerificationErrorThreshold       = 3
)

func accountsStateNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.StateNode(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeAccount}).
		Pipe(
			pipeline.StateNode(&horizonProcessors.DatabaseProcessor{
				AccountsQ:     q,
				Action:        horizonProcessors.Accounts,
				IngestVersion: CurrentVersion,
			}),
			pipeline.StateNode(&horizonProcessors.DatabaseProcessor{
				SignersQ:      q,
				Action:        horizonProcessors.AccountsForSigner,
				IngestVersion: CurrentVersion,
			}),
		)
}

func dataDBStateNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.StateNode(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeData}).
		Pipe(
			pipeline.StateNode(&horizonProcessors.DatabaseProcessor{
				DataQ:         q,
				Action:        horizonProcessors.Data,
				IngestVersion: CurrentVersion,
			}),
		)
}

func orderBookDBStateNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.StateNode(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeOffer}).
		Pipe(
			pipeline.StateNode(&horizonProcessors.DatabaseProcessor{
				OffersQ:       q,
				Action:        horizonProcessors.Offers,
				IngestVersion: CurrentVersion,
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

func trustLinesDBStateNode(q *history.Q) *supportPipeline.PipelineNode {
	return pipeline.StateNode(&processors.EntryTypeFilter{Type: xdr.LedgerEntryTypeTrustline}).
		Pipe(
			pipeline.StateNode(&horizonProcessors.DatabaseProcessor{
				TrustLinesQ:   q,
				AssetStatsQ:   q,
				Action:        horizonProcessors.TrustLines,
				IngestVersion: CurrentVersion,
			}),
		)
}

func buildStatePipeline(historyQ *history.Q, graph *orderbook.OrderBookGraph) *pipeline.StatePipeline {
	statePipeline := &pipeline.StatePipeline{}

	statePipeline.SetRoot(
		pipeline.StateNode(&processors.RootProcessor{}).
			Pipe(
				accountsStateNode(historyQ),
				dataDBStateNode(historyQ),
				orderBookDBStateNode(historyQ),
				orderBookGraphStateNode(graph),
				trustLinesDBStateNode(historyQ),
			),
	)

	return statePipeline
}

func orderBookGraphLedgerNode(graph *orderbook.OrderBookGraph) *supportPipeline.PipelineNode {
	return pipeline.LedgerNode(&horizonProcessors.OrderbookProcessor{
		OrderBookGraph: graph,
	})
}

func buildLedgerPipeline(historyQ *history.Q, graph *orderbook.OrderBookGraph) *pipeline.LedgerPipeline {
	ledgerPipeline := &pipeline.LedgerPipeline{}

	ledgerPipeline.SetRoot(
		pipeline.LedgerNode(&processors.RootProcessor{}).
			Pipe(
				// This subtree will only run when `IngestUpdateDatabase` is set.
				pipeline.LedgerNode(&horizonProcessors.ContextFilter{horizonProcessors.IngestUpdateDatabase}).
					Pipe(
						pipeline.LedgerNode(&horizonProcessors.DatabaseProcessor{
							AccountsQ:     historyQ,
							DataQ:         historyQ,
							OffersQ:       historyQ,
							SignersQ:      historyQ,
							TrustLinesQ:   historyQ,
							AssetStatsQ:   historyQ,
							LedgersQ:      historyQ,
							Action:        horizonProcessors.All,
							IngestVersion: CurrentVersion,
						}),
					),
				orderBookGraphLedgerNode(graph),
			),
	)

	return ledgerPipeline
}

func preProcessingHook(
	ctx context.Context,
	pipelineType pType,
	system *System,
	historySession *db.Session,
) (context.Context, error) {
	var err error
	defer func() {
		// Rollback a transaction if pre hook returns errors. Without it all
		// queries will fail indefinietly with a following error:
		// current transaction is aborted, commands ignored until end of transaction block
		if err != nil {
			historySession.Rollback()
		}
	}()

	historyQ := &history.Q{historySession}

	// Start a transaction only if not in a transaction already.
	// The only case this can happen is during the first run when
	// a transaction is started to get the latest ledger `FOR UPDATE`
	// in `System.Run()`.
	if tx := historySession.GetTx(); tx == nil {
		err = historySession.Begin()
		if err != nil {
			return ctx, errors.Wrap(err, "Error starting a transaction")
		}
	}

	// We need to get this value `FOR UPDATE` so all other instances
	// are blocked.
	lastIngestedLedger, err := historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return ctx, errors.Wrap(err, "Error getting last ledger")
	}

	ledgerSeq := pipeline.GetLedgerSequenceFromContext(ctx)

	updateDatabase := false
	if pipelineType == statePipeline {
		// State pipeline is always fully run because loading offers
		// from a database is done outside the pipeline.
		updateDatabase = true
	} else {
		// mark the system as ready because we have progressed to running
		// the ledger pipeline
		system.setStateReady()

		if lastIngestedLedger+1 == ledgerSeq {
			// lastIngestedLedger+1 == ledgerSeq what means that this instance
			// is the main ingesting instance in this round and should update a
			// database.
			updateDatabase = true
			ctx = context.WithValue(ctx, horizonProcessors.IngestUpdateDatabase, true)
		}
	}

	// If we are not going to update a DB release a lock by rolling back a
	// transaction.
	if !updateDatabase {
		historySession.Rollback()
	}

	log.WithFields(logpkg.F{
		"ledger":            ledgerSeq,
		"type":              pipelineType,
		"updating_database": updateDatabase,
	}).Info("Processing ledger")

	return ctx, nil
}

func postProcessingHook(
	ctx context.Context,
	err error,
	pipelineType pType,
	system *System,
	graph *orderbook.OrderBookGraph,
	historySession *db.Session,
) error {
	defer historySession.Rollback()
	defer graph.Discard()
	historyQ := &history.Q{historySession}
	isMaster := false

	ledgerSeq := pipeline.GetLedgerSequenceFromContext(ctx)

	if err != nil {
		if err == supportPipeline.ErrShutdown {
			return nil
		}

		switch errors.Cause(err).(type) {
		case verify.StateError:
			markStateInvalid(historySession, err)
		default:
			log.
				WithFields(logpkg.F{
					"ledger": ledgerSeq,
					"type":   pipelineType,
					"err":    err,
				}).
				Error("Error processing ledger")
		}
		return err
	}

	if tx := historySession.GetTx(); tx != nil {
		isMaster = true

		// If we're in a transaction we're updating database with new data.
		// We get lastIngestedLedger from a DB here to do an extra check
		// if the current node should really be updating a DB.
		// This is "just in case" if lastIngestedLedger is not selected
		// `FOR UPDATE` due to a bug or accident. In such case we error and
		// rollback.
		var lastIngestedLedger uint32
		lastIngestedLedger, err = historyQ.GetLastLedgerExpIngest()
		if err != nil {
			return errors.Wrap(err, "Error getting last ledger")
		}

		if lastIngestedLedger != 0 && lastIngestedLedger+1 != ledgerSeq {
			return errors.New("The local latest sequence is not equal to global sequence + 1")
		}

		if err = historyQ.UpdateLastLedgerExpIngest(ledgerSeq); err != nil {
			return errors.Wrap(err, "Error updating last ingested ledger")
		}

		if err = historyQ.UpdateExpIngestVersion(CurrentVersion); err != nil {
			return errors.Wrap(err, "Error updating expingest version")
		}

		if err = historySession.Commit(); err != nil {
			return errors.Wrap(err, "Error commiting db transaction")
		}
	}

	err = graph.Apply(ledgerSeq)
	if err != nil {
		return errors.Wrap(err, "Error applying order book changes")
	}

	stateInvalid, err := historyQ.GetExpStateInvalid()
	if err != nil {
		log.WithField("err", err).Error("Error getting state invalid value")
	}

	// Run verification routine only when...
	if system != nil && // system is defined (not in tests)...
		!stateInvalid && // state has not been proved to be invalid...
		!system.disableStateVerification && // state verification is not disabled...
		pipelineType == ledgerPipeline && // it's a ledger pipeline...
		isMaster && // it's a master ingestion node (to verify on a single node only)...
		historyarchive.IsCheckpoint(ledgerSeq) { // it's a checkpoint ledger.
		system.wg.Add(1)
		go func(offerEntries []xdr.OfferEntry) {
			defer system.wg.Done()
			graphOffers := map[xdr.Int64]xdr.OfferEntry{}
			for _, entry := range offerEntries {
				graphOffers[entry.OfferId] = entry
			}

			err := system.verifyState(graphOffers)
			if err != nil {
				errorCount := system.incrementStateVerificationErrors()
				switch errors.Cause(err).(type) {
				case verify.StateError:
					markStateInvalid(historySession, err)
				default:
					logger := log.WithField("err", err).Warn
					if errorCount >= stateVerificationErrorThreshold {
						logger = log.WithField("err", err).Error
					}
					logger("State verification errored")
				}
			} else {
				system.resetStateVerificationErrors()
			}
		}(graph.Offers())
	}

	log.WithFields(logpkg.F{"ledger": ledgerSeq, "type": pipelineType}).Info("Processed ledger")
	return nil
}

func markStateInvalid(historySession *db.Session, err error) {
	log.WithField("err", err).Error("STATE IS INVALID!")
	q := &history.Q{historySession.Clone()}
	if err := q.UpdateExpStateInvalid(true); err != nil {
		log.WithField("err", err).Error("Error updating state invalid value")
	}
}

func addPipelineHooks(
	system *System,
	p supportPipeline.PipelineInterface,
	historySession *db.Session,
	ingestSession ingest.Session,
	graph *orderbook.OrderBookGraph,
) {
	var pipelineType pType
	switch p.(type) {
	case *pipeline.StatePipeline:
		pipelineType = statePipeline
	case *pipeline.LedgerPipeline:
		pipelineType = ledgerPipeline
	default:
		panic(fmt.Sprintf("Unknown pipeline type %T", p))
	}

	p.AddPreProcessingHook(func(ctx context.Context) (context.Context, error) {
		return preProcessingHook(ctx, pipelineType, system, historySession)
	})

	p.AddPostProcessingHook(func(ctx context.Context, err error) error {
		return postProcessingHook(ctx, err, pipelineType, system, graph, historySession)
	})
}
