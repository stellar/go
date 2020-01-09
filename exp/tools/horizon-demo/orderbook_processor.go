package main

import (
	"context"
	"fmt"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/xdr"
)

// OrderbookProcessor is a processor (both state and ledger) that's responsible
// for updating orderbook graph with new/updated/removed offers. Orderbook graph
// can be later used for path finding.
type OrderbookProcessor struct {
	OrderBookGraph *orderbook.OrderBookGraph
}

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

func (p *OrderbookProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) error {
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

		if transaction.Result.Result.Result.Code != xdr.TransactionResultCodeTxSuccess {
			continue
		}

		changes, err := transaction.GetChanges()
		if err != nil {
			return err
		}

		for _, change := range changes {
			if change.Type != xdr.LedgerEntryTypeOffer {
				continue
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

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	return nil
}

func (p *OrderbookProcessor) Name() string {
	return fmt.Sprintf("OrderbookProcessor")
}

func (p *OrderbookProcessor) Reset() {}

var _ ingestpipeline.StateProcessor = &OrderbookProcessor{}
var _ ingestpipeline.LedgerProcessor = &OrderbookProcessor{}
