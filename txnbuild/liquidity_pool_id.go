//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package txnbuild

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LiquidityPoolId represents the Stellar liquidity pool id.
type LiquidityPoolId [32]byte

func NewLiquidityPoolId(a, b Asset) (LiquidityPoolId, error) {
	assets := Assets([]Asset{a, b})
	sort.Sort(assets)
	if assets[0] != a {
		return LiquidityPoolId{}, fmt.Errorf("AssetA must be <= AssetB")
	}

	xdrAssetA, err := assets[0].ToXDR()
	if err != nil {
		return LiquidityPoolId{}, errors.Wrap(err, "failed to build XDR AssetA ID")
	}

	xdrAssetB, err := assets[1].ToXDR()
	if err != nil {
		return LiquidityPoolId{}, errors.Wrap(err, "failed to build XDR AssetB ID")
	}

	params := xdr.LiquidityPoolConstantProductParameters{
		AssetA: xdrAssetA,
		AssetB: xdrAssetB,
		// Hardcoded for now, as it must be this.
		Fee: xdr.LiquidityPoolFeeV18,
	}

	buf := &bytes.Buffer{}
	_, err = xdr.Marshal(buf, params)
	if err != nil {
		return LiquidityPoolId{}, errors.Wrap(err, "failed to build liquidity pool id")
	}
	return sha256.Sum256(buf.Bytes()), nil
}

func (lpi LiquidityPoolId) ToXDR() (xdr.PoolId, error) {
	return xdr.PoolId(lpi), nil
}

func liquidityPoolIdFromXDR(poolId xdr.PoolId) (LiquidityPoolId, error) {
	return LiquidityPoolId(poolId), nil
}
