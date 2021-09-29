package orderbook

import (
	"sort"

	"github.com/stellar/go/xdr"
)

// edgeSet maintains a mapping of strings (asset keys) to a set of venues, which
// is composed of a sorted lists of offers and, optionally, a liquidity pool.
// The offers are sorted by ascending price (in terms of the buying asset).
type edgeSet []edge

type edge struct {
	key   string
	value Venues
}

func (e edgeSet) find(key string) int {
	for i := 0; i < len(e); i++ {
		if e[i].key == key {
			return i
		}
	}
	return -1
}

// addOffer will insert the given offer into the edge set
func (e edgeSet) addOffer(key string, offer xdr.OfferEntry) edgeSet {
	// The list of offers in a venue is sorted by cheapest to most expensive
	// price to convert buyingAsset to sellingAsset
	i := e.find(key)
	if i < 0 {
		return append(e, edge{key: key, value: Venues{offers: []xdr.OfferEntry{offer}}})
	}

	edge := e[i]
	// find the smallest i such that Price of offers[i] >  Price of offer
	insertIndex := sort.Search(len(edge.value.offers), func(j int) bool {
		return offer.Price.Cheaper(edge.value.offers[j].Price)
	})

	// then insert it into the slice (taken from Method 2 at
	// https://github.com/golang/go/wiki/SliceTricks#insert).
	offers := append(edge.value.offers, xdr.OfferEntry{}) // add to end
	copy(offers[insertIndex+1:], offers[insertIndex:])    // shift right by 1
	offers[insertIndex] = offer                           // insert

	e[i].value = Venues{offers: offers, pool: edge.value.pool}
	return e
}

// addPool makes `pool` a viable venue at `key`.
func (e edgeSet) addPool(key string, pool xdr.LiquidityPoolEntry) edgeSet {
	i := e.find(key)
	if i < 0 {
		return append(e, edge{key: key, value: Venues{pool: pool}})
	}
	e[i].value.pool = pool
	return e
}

// removeOffer will delete the given offer from the edge set, returning whether
// or not the given offer was actually found.
func (e edgeSet) removeOffer(key string, offerID xdr.Int64) (edgeSet, bool) {
	i := e.find(key)
	if i < 0 {
		return e, false
	}

	edge := e[i]
	offers := edge.value.offers
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
		return e, false
	}

	if len(updatedOffers) == 0 && edge.value.pool.Body.ConstantProduct == nil {
		return append(e[:i], e[i+1:]...), true
	}
	e[i].value.offers = updatedOffers
	return e, true
}

func (e edgeSet) removePool(key string) edgeSet {
	i := e.find(key)
	if i < 0 {
		return e
	}

	if len(e[i].value.offers) == 0 {
		return append(e[:i], e[i+1:]...)
	}

	e[i].value = Venues{offers: e[i].value.offers}
	return e
}
