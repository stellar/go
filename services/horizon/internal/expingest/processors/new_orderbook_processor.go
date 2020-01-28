package processors

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// OrderbookProcessor is a processor (both state and ledger) that's responsible
// for updating orderbook graph with new/updated/removed offers. Orderbook graph
// can be later used for path finding.
type OrderbookProcessor struct {
	graph *orderbook.OrderBookGraph

	cache *io.LedgerEntryChangeCache
}

func NewOrderbookProcessor(graph *orderbook.OrderBookGraph) *OrderbookProcessor {
	return &OrderbookProcessor{
		graph: graph,
		cache: io.NewLedgerEntryChangeCache(),
	}
}

func (p *OrderbookProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeOffer {
		return nil
	}

	err := p.cache.AddChange(change)
	if err != nil {
		return errors.Wrap(err, "error adding to ledgerCache")
	}

	return nil
}

func (p *OrderbookProcessor) Commit() error {
	changes := p.cache.GetChanges()
	for _, change := range changes {
		switch {
		case change.Post != nil:
			// Created or updated
			offer := change.Post.Data.MustOffer()
			p.graph.AddOffer(offer)
		case change.Pre != nil && change.Post == nil:
			// Removed
			offer := change.Pre.Data.MustOffer()
			p.graph.RemoveOffer(offer.OfferId)
		}
	}
	return nil
}
