package xdr

import (
	"bytes"
	"crypto/sha256"

	"github.com/stellar/go/support/errors"
)

func NewPoolId(a, b Asset, fee Int32) (PoolId, error) {
	if b.LessThan(a) {
		return PoolId{}, errors.New("asset A must be < asset B")
	}

	// Assume the assets are already sorted.
	params := LiquidityPoolParameters{
		Type: LiquidityPoolTypeLiquidityPoolConstantProduct,
		ConstantProduct: &LiquidityPoolConstantProductParameters{
			AssetA: a,
			AssetB: b,
			Fee:    fee,
		},
	}

	buf := &bytes.Buffer{}
	if _, err := Marshal(buf, params); err != nil {
		return PoolId{}, errors.Wrap(err, "failed to build liquidity pool id")
	}
	return sha256.Sum256(buf.Bytes()), nil
}
