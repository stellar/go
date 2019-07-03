package orderbook

import (
	"math/big"
	"sort"

	"github.com/stellar/go/xdr"
)

// edgeSet maps an asset to all offers which buy that asset
// note that each key in the map is obtained by calling offer.Buying.String()
// also, the offers are sorted by price (in terms of the buying asset)
// from cheapest to expensive
type edgeSet map[string][]xdr.OfferEntry

// add will insert the given offer into the edge set
func (e edgeSet) add(offer xdr.OfferEntry) {
	buyingAsset := offer.Buying.String()
	// offers is sorted by cheapest to most expensive price to convert buyingAsset to sellingAsset
	offers := e[buyingAsset]
	if len(offers) == 0 {
		e[buyingAsset] = []xdr.OfferEntry{offer}
		return
	}

	// find the smallest i such that Price of offers[i] >  Price of offer
	insertIndex := sort.Search(len(offers), func(i int) bool {
		return big.NewRat(int64(offers[i].Price.N), int64(offers[i].Price.D)).
			Cmp(big.NewRat(int64(offer.Price.N), int64(offer.Price.D))) > 0
	})

	offers = append(offers, offer)
	last := len(offers) - 1
	for insertIndex < last {
		offers[insertIndex], offers[last] = offers[last], offers[insertIndex]
		insertIndex++
	}
	e[buyingAsset] = offers
}

// remove will delete the given offer from the edge set
// buyingAsset is obtained by calling offer.Buying.String()
func (e edgeSet) remove(offerID xdr.Int64, buyingAsset string) bool {
	edges := e[buyingAsset]
	if len(edges) == 0 {
		return false
	}
	contains := false

	for i := 0; i < len(edges); i++ {
		if edges[i].OfferId == offerID {
			contains = true
			for j := i + 1; j < len(edges); j++ {
				edges[i] = edges[j]
				i++
			}
			edges = edges[0 : len(edges)-1]
			break
		}
	}
	if contains {
		if len(edges) == 0 {
			delete(e, buyingAsset)
		} else {
			e[buyingAsset] = edges
		}
	}

	return contains
}
