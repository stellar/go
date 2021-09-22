package orderbook

import (
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
