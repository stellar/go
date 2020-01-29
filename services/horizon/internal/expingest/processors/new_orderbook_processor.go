package processors

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// NewOrderbookProcessor is a processor (both state and ledger) that's responsible
// for updating orderbook graph with new/updated/removed offers. Orderbook graph
// can be later used for path finding.
type NewOrderbookProcessor struct {
	OrderBookGraph *orderbook.OrderBookGraph

	cache *io.LedgerEntryChangeCache
}

func (p *NewOrderbookProcessor) Init(header xdr.LedgerHeader) error {
	p.cache = io.NewLedgerEntryChangeCache()
	return nil
}

func (p *NewOrderbookProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeOffer {
		return nil
	}

	err := p.cache.AddChange(change)
	if err != nil {
		return errors.Wrap(err, "error adding to ledgerCache")
	}

	return nil
}

func (p *NewOrderbookProcessor) Commit() error {
	changes := p.cache.GetChanges()
	for _, change := range changes {
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
	return nil
}
