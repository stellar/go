package processors

import (
	"context"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
)

// TransactionFilterProcessor is a processor which can be configured to filter away failed transactions
type TransactionFilterProcessor struct {
	IngestFailedTransactions bool
}

func (p *TransactionFilterProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer w.Close()

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

		if p.IngestFailedTransactions || transaction.Successful() {
			err = w.Write(transaction)
			if err != nil {
				if err == stdio.ErrClosedPipe {
					// Reader does not need more data
					return nil
				}
				return err
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	return nil
}

func (p *TransactionFilterProcessor) Name() string {
	return "TransactionFilterProcessor"
}

func (p *TransactionFilterProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &TransactionFilterProcessor{}
