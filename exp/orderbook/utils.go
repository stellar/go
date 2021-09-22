package orderbook

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

// contains searches for a string needle in a haystack
func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// getPoolAssets retrieves string representations of a pool's reserves
func getPoolAssets(pool xdr.LiquidityPoolEntry) (string, string) {
	params := pool.Body.MustConstantProduct().Params
	return params.AssetA.String(), params.AssetB.String()
}

// makeTradingPair easily turns two assets into a string trading pair
func makeTradingPair(buying, selling xdr.Asset) tradingPair {
	return tradingPair{buyingAsset: buying.String(), sellingAsset: selling.String()}
}

// The below helpers are only useful for testing and/or debugging.

func getCode(asset xdr.Asset) string {
	code := asset.GetCode()
	if code == "" {
		return "xlm"
	}
	return code
}

func printOffers(offers ...xdr.OfferEntry) {
	for i, offer := range offers {
		fmt.Printf("  %d - offering %d %s for %s @ %s each (id=%d)\n", i, offer.Amount,
			getCode(offer.Selling), getCode(offer.Buying), offer.Price, offer.OfferId)
	}
}

func printPath(path Path) {
	fmt.Printf(" - %d %s -> ", path.SourceAmount, getCode(path.SourceAsset))

	for _, hop := range path.InteriorNodes {
		fmt.Printf("%s -> ", getCode(hop))
	}

	fmt.Printf("%d %s\n",
		path.DestinationAmount, getCode(path.DestinationAsset))
}

func makeVenues(offers ...xdr.OfferEntry) Venues {
	return Venues{offers: offers}
}
