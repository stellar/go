package orderbook

import (
	"sort"

	"github.com/stellar/go/xdr"
)

// edgeSet maintains a mapping of strings (asset keys) to a set of venues, which
// is composed of a sorted lists of offers and, optionally, a liquidity pool.
// The offers are sorted by ascending price (in terms of the buying asset).
type edgeSet map[string]Venues

// addOffer will insert the given offer into the edge set
func (e edgeSet) addOffer(key string, offer xdr.OfferEntry) {
	// The list of offers in a venue is sorted by cheapest to most expensive
	// price to convert buyingAsset to sellingAsset
	venues := e[key]
	if len(venues.offers) == 0 {
		e[key] = Venues{
			offers: []xdr.OfferEntry{offer},
			pool:   venues.pool,
		}
		return
	}

	// find the smallest i such that Price of offers[i] >  Price of offer
	insertIndex := sort.Search(len(venues.offers), func(i int) bool {
		return offer.Price.Cheaper(venues.offers[i].Price)
	})

	// then insert it into the slice (taken from Method 2 at
	// https://github.com/golang/go/wiki/SliceTricks#insert).
	offers := append(venues.offers, xdr.OfferEntry{})  // add to end
	copy(offers[insertIndex+1:], offers[insertIndex:]) // shift right by 1
	offers[insertIndex] = offer                        // insert

	e[key] = Venues{offers: offers, pool: venues.pool}
}

// addPool makes `pool` a viable venue at `key`.
func (e edgeSet) addPool(key string, pool xdr.LiquidityPoolEntry) {
	venues := e[key]
	venues.pool = pool
	e[key] = venues
}

// removeOffer will delete the given offer from the edge set, returning whether
// or not the given offer was actually found.
func (e edgeSet) removeOffer(key string, offerID xdr.Int64) bool {
	venues := e[key]
	offers := venues.offers

	contains := false
	for i, offer := range offers {
		if offer.OfferId != offerID {
			continue
		}

		// remove the entry in the slice at this location (taken from
		// https://github.com/golang/go/wiki/SliceTricks#cut).
		offers = append(offers[:i], offers[i+1:]...)
		contains = true
		break
	}

	if !contains {
		return false
	}

	if len(offers) == 0 && venues.pool.Body.ConstantProduct == nil {
		delete(e, key)
	} else {
		venues.offers = offers
		e[key] = venues
	}

	return true
}

func (e edgeSet) removePool(key string) {
	e[key] = Venues{offers: e[key].offers}
}

func (e edgeSet) isEmpty(key string) bool {
	return len(e[key].offers) == 0 && e[key].pool.Body.ConstantProduct == nil
}
