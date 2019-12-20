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

	// Process operations
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

		if err = operationBatch.Add(transaction, sequence); err != nil {
			return errors.Wrap(err, "Error batch inserting operation rows")
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

	// use an older lookup sequence because the experimental ingestion system and the
	// legacy ingestion system might not be in sync
	checkSequence := int32(sequence - 10)
	var valid bool
	valid, err = p.OperationsQ.CheckExpOperations(checkSequence)
	if err != nil {
		log.WithField("sequence", checkSequence).WithError(err).
			Error("Could not compare operations for ledger")
		return nil
	}

	if !valid {
		log.WithField("sequence", checkSequence).
			Error("rows for ledger in exp_history_operations does not match " +
				"operations in history_operations")
	}

	return nil
}

// Name processor name
func (p *OperationProcessor) Name() string {
	return "OperationProcessor"
}

// Reset resets processor
func (p *OperationProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &OperationProcessor{}
