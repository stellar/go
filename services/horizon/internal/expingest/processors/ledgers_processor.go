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

type LedgersProcessor struct {
	LedgersQ      history.QLedgers
	IngestVersion int
}

func (p *LedgersProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
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
	if v := ctx.Value(IngestUpdateDatabase); !(v != nil && v.(bool)) {
		return nil
	}

	var successTxCount, failedTxCount, opCount int

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

		if transaction.Successful() {
			successTxCount++
			opCount += len(transaction.Envelope.Tx.Operations)
		} else {
			failedTxCount++
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	rowsAffected, err := p.LedgersQ.InsertLedger(
		r.GetHeader(),
		successTxCount,
		failedTxCount,
		opCount,
		p.IngestVersion,
	)
	if err != nil {
		return errors.Wrap(err, "Could not insert ledger")
	}
	if rowsAffected != 1 {
		log.WithField("rowsAffected", rowsAffected).
			WithField("sequence", r.GetSequence()).
			Error("Invalid number of rows affected when ingesting new ledger")
		return errors.Errorf(
			"0 rows affected when ingesting new ledger: %v",
			r.GetSequence(),
		)
	}

	return nil
}

func (p *LedgersProcessor) Name() string {
	return "LedgersProcessor"
}

func (p *LedgersProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &LedgersProcessor{}
