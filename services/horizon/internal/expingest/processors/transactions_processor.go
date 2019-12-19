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

type TransactionProcessor struct {
	TransactionsQ history.QTransactions
}

func (p *TransactionProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
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

	transactionBatch := p.TransactionsQ.NewTransactionBatchInsertBuilder(maxBatchSize)
	sequence := r.GetSequence()

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

		if err = transactionBatch.Add(transaction, sequence); err != nil {
			return errors.Wrap(err, "Error batch inserting transaction rows")
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	if err = transactionBatch.Exec(); err != nil {
		return errors.Wrap(err, "Error flushing transaction batch")
	}

	// use an older lookup sequence because the experimental ingestion system and the
	// legacy ingestion system might not be in sync
	checkSequence := int32(sequence - 10)
	var valid bool
	valid, err = p.TransactionsQ.CheckExpTransactions(checkSequence)
	if err != nil {
		log.WithField("sequence", checkSequence).WithError(err).
			Error("Could not compare transactions for ledger")
		return nil
	}

	if !valid {
		log.WithField("sequence", checkSequence).
			Error("rows for ledger in exp_history_transactions does not match " +
				"transactions in history_transactions")
	}

	return nil
}

func (p *TransactionProcessor) Name() string {
	return "TransactionProcessor"
}

func (p *TransactionProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &TransactionProcessor{}
