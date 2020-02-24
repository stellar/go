package processors

import (
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestProcessOrderBookState(t *testing.T) {
	graph := orderbook.NewOrderBookGraph()
	processor := NewOrderbookProcessor(graph)

	err := processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(1),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 2},
				},
			},
		},
	})
	assert.NoError(t, err)
	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(2),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 2},
				},
			},
		},
	})
	assert.NoError(t, err)
	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(3),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 2},
				},
			},
		},
	})
	assert.NoError(t, err)

	assert.NoError(t, processor.Commit())
	if err := graph.Apply(2); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	expectedOffers := map[xdr.Int64]bool{
		xdr.Int64(1): true,
		xdr.Int64(2): true,
		xdr.Int64(3): true,
	}

	offers := graph.Offers()
	for _, offer := range offers {
		if !expectedOffers[offer.OfferId] {
			t.Fatalf("unexpected offer id %v", offer.OfferId)
		}
		delete(expectedOffers, offer.OfferId)
	}
	if len(expectedOffers) != 0 {
		t.Fatal("expected offers does not match offers in graph")
	}
}

func TestProcessOrderBookLedger(t *testing.T) {
	graph := orderbook.NewOrderBookGraph()
	processor := NewOrderbookProcessor(graph)

	// should be ignored because it's not an offer type
	err := processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
	})
	assert.NoError(t, err)

	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(1),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 2},
				},
			},
		},
	})
	assert.NoError(t, err)

	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(2),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 3},
				},
			},
		},
	})
	assert.NoError(t, err)

	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(3),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{3, 1},
				},
			},
		},
	})
	assert.NoError(t, err)

	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(2),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 3},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(2),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 6},
				},
			},
		},
	})
	assert.NoError(t, err)

	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(3),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{3, 1},
				},
			},
		},
		Post: nil,
	})
	assert.NoError(t, err)

	assert.NoError(t, processor.Commit())

	if err := graph.Apply(2); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	expectedOffers := map[xdr.Int64]xdr.Price{
		xdr.Int64(1): xdr.Price{1, 2},
		xdr.Int64(2): xdr.Price{1, 6},
	}

	offers := graph.Offers()
	for _, offer := range offers {
		if price, ok := expectedOffers[offer.OfferId]; !ok {
			t.Fatalf("unexpected offer id %v", offer.OfferId)
		} else if offer.Price != price {
			t.Fatalf("unexpected offer price %v for offer with id %v", offer.Price, offer.OfferId)
		}
		delete(expectedOffers, offer.OfferId)
	}
	if len(expectedOffers) != 0 {
		t.Fatal("expected offers does not match offers in graph")
	}
}

func TestProcessOrderBookLedgerProcessUpgradeChanges(t *testing.T) {
	graph := orderbook.NewOrderBookGraph()
	processor := NewOrderbookProcessor(graph)

	err := processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(1),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 2},
				},
			},
		},
	})
	assert.NoError(t, err)

	err = processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(1),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{1, 2},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					OfferId:  xdr.Int64(1),
					SellerId: xdr.MustAddress("GCFMARUTEFR6NW5HU5JZIVHEN5MO764GQKGRLHOIRY265Z343NZ7AK3J"),
					Price:    xdr.Price{100, 2},
				},
			},
		},
	})
	assert.NoError(t, err)

	assert.NoError(t, processor.Commit())

	if err := graph.Apply(2); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	offers := graph.Offers()
	assert.Equal(t, xdr.Int32(100), offers[0].Price.N)
	assert.Equal(t, xdr.Int32(2), offers[0].Price.D)
}
