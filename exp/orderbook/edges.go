package orderbook

import (
	"sort"

	"github.com/stellar/go/xdr"
)

// edgeSet maintains a mapping of strings (asset keys) to a set of venues, which
// is composed of a sorted lists of offers and, optionally, a liquidity pool.
// The offers are sorted by ascending price (in terms of the buying asset).
type edgeSet struct {
	keys   []string
	values []Venues
}

func findKey(keys []string, key string) int {
	for i := 0; i < len(keys); i++ {
		if keys[i] == key {
			return i
		}
	}
	return -1
}

// addOffer will insert the given offer into the edge set
func (e *edgeSet) addOffer(key string, offer xdr.OfferEntry) {
	// The list of offers in a venue is sorted by cheapest to most expensive
	// price to convert buyingAsset to sellingAsset
	i := findKey(e.keys, key)
	if i < 0 {
		e.keys = append(e.keys, key)
		e.values = append(e.values, Venues{
			offers: []xdr.OfferEntry{offer},
		})
		return
	}

	venues := e.values[i]
	// find the smallest i such that Price of offers[i] >  Price of offer
	insertIndex := sort.Search(len(venues.offers), func(i int) bool {
		return offer.Price.Cheaper(venues.offers[i].Price)
	})

	// then insert it into the slice (taken from Method 2 at
	// https://github.com/golang/go/wiki/SliceTricks#insert).
	offers := append(venues.offers, xdr.OfferEntry{})  // add to end
	copy(offers[insertIndex+1:], offers[insertIndex:]) // shift right by 1
	offers[insertIndex] = offer                        // insert

	e.values[i] = Venues{offers: offers, pool: venues.pool}
}

// addPool makes `pool` a viable venue at `key`.
func (e *edgeSet) addPool(key string, pool xdr.LiquidityPoolEntry) {
	i := findKey(e.keys, key)
	if i < 0 {
		e.keys = append(e.keys, key)
		e.values = append(e.values, Venues{
			pool: pool,
		})
		return
	}
	e.values[i].pool = pool
}

// removeOffer will delete the given offer from the edge set, returning whether
// or not the given offer was actually found.
func (e *edgeSet) removeOffer(key string, offerID xdr.Int64) bool {
	i := findKey(e.keys, key)
	if i < 0 {
		return false
	}

	offers := e.values[i].offers
	updatedOffers := offers
	contains := false
	for i, offer := range offers {
		if offer.OfferId != offerID {
			continue
		}

		// remove the entry in the slice at this location (taken from
		// https://github.com/golang/go/wiki/SliceTricks#cut).
		updatedOffers = append(offers[:i], offers[i+1:]...)
		contains = true
		break
	}

	if !contains {
		return false
	}

	if len(updatedOffers) == 0 && e.values[i].pool.Body.ConstantProduct == nil {
		e.values = append(e.values[:i], e.values[i+1:]...)
		e.keys = append(e.keys[:i], e.keys[i+1:]...)
	} else {
		e.values[i].offers = updatedOffers
	}

	return true
}

func (e *edgeSet) removePool(key string) {
	i := findKey(e.keys, key)
	if i < 0 {
		return
	}

	if len(e.values[i].offers) == 0 {
		e.values = append(e.values[:i], e.values[i+1:]...)
		e.keys = append(e.keys[:i], e.keys[i+1:]...)
		return
	}

	e.values[i] = Venues{offers: e.values[i].offers}
}
