package processors

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (p *OrderbookProcessor) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReader, w io.StateWriter) error {
	defer r.Close()
	defer w.Close()

	for {
		entryChange, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		offer := entryChange.MustState().Data.MustOffer()
		p.OrderBookGraph.AddOffer(offer)

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	return nil
}

func (p *OrderbookProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer w.Close()

	// Exit early if not ingesting state (history catchup). The filtering in parent
	// processor should do it, unfortunately it won't work in case of meta upgrades.
	// Should be fixed after ingest refactoring.
	if v := ctx.Value(IngestUpdateState); !(v != nil && v.(bool)) {
		return nil
	}

	for {
		transaction, err := r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		if !transaction.Successful() {
			continue
		}

		changes, err := transaction.GetChanges()
		if err != nil {
			return errors.Wrap(err, "Error in transaction.GetChanges()")
		}
		for _, change := range changes {
			p.processChange(change)
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

func (p *OrderbookProcessor) processChange(change io.Change) {
	if change.Type != xdr.LedgerEntryTypeOffer {
		return
	}

	switch {
	case change.Post != nil:
		// Created or updated
		offer := change.Post.Data.MustOffer()
		p.OrderBookGraph.AddOffer(offer)
	case change.Pre != nil && change.Post == nil:
		// Removed
		offer := change.Pre.Data.MustOffer()
		p.OrderBookGraph.RemoveOffer(offer.OfferId)
	}
}

func (p *OrderbookProcessor) Name() string {
	return fmt.Sprintf("OrderbookProcessor")
}

func (p *OrderbookProcessor) Reset() {}

var _ ingestpipeline.StateProcessor = &OrderbookProcessor{}
var _ ingestpipeline.LedgerProcessor = &OrderbookProcessor{}
