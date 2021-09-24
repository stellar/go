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

func max(vals ...xdr.Int64) xdr.Int64 {
	best := vals[0]
	for _, v := range vals[1:] {
		if v >= best {
			best = v
		}
	}
	return best
}

// positiveMin returns the smallest positive value possible.
func positiveMin(vals ...xdr.Int64) xdr.Int64 {
	best := vals[0]
	for _, v := range vals[1:] {
		if v <= 0 {
			continue
		}
		if v < best || best <= 0 {
			best = v
		}
	}
	return best
}
