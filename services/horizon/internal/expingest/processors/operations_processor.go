package processors

import (
	"context"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
)

// OperationProcessor operations processor
type OperationProcessor struct {
	OperationsQ history.QOperations
}

// ProcessLedger process the given ledger
func (p *OperationProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer func() {
		// io.LedgerReader.Close() returns error if upgrade changes have not
		// been processed so it's worth checking the error.
		closeErr := r.Close()
		// Do not overwrite the previous error
		if err == nil {
			err = closeErr
		}
	}()
	defer w.Close()
	r.IgnoreUpgradeChanges()

	// Exit early if not ingesting into a DB
	if v := ctx.Value(IngestUpdateDatabase); v == nil {
		return nil
	}

	operationBatch := p.OperationsQ.NewOperationBatchInsertBuilder(maxBatchSize)
	sequence := r.GetSequence()

	// Process transaction meta
	for {
		var transaction io.LedgerTransaction
		transaction, err = r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		for i, op := range transaction.Envelope.Tx.Operations {
			operation := history.TransactionOperation{
				Index:          uint32(i),
				Transaction:    transaction,
				Operation:      op,
				LedgerSequence: sequence,
			}

			if err = operationBatch.Add(operation); err != nil {
				return errors.Wrap(err, "Error batch inserting operation rows")
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	if err = operationBatch.Exec(); err != nil {
		return errors.Wrap(err, "Error flushing operation batch")
	}

	// TODO add verifier

	return nil
}

// Name processor name
func (p *OperationProcessor) Name() string {
	return "OperationProcessor"
}

// Reset resets processor
func (p *OperationProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &OperationProcessor{}
