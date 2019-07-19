package processors

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
)

func (p *LedgerFilter) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) error {
	defer r.Close()
	defer w.Close()

	for {
		transaction, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if r.GetSequence() < p.IgnoreLedgersBefore {
			continue
		}

		if err := w.Write(transaction); err != nil {
			if err == stdio.ErrClosedPipe {
				// Reader does not need more data
				return nil
			}
			return err
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

func (p *LedgerFilter) Name() string {
	return fmt.Sprintf("LedgerFilter (%v)", p.IgnoreLedgersBefore)
}

var _ ingestpipeline.LedgerProcessor = &LedgerFilter{}
