// This implementation is less performant than mul_bits.go
// however, mul_bits.go is only compatible with go 1.12+ and
// this implementation will work on go 1.10 and 1.9
//
// +build !go1.12

package price

import (
	"fmt"
	"math/big"
)

// MulFractionRoundDown sets x = (x * n) / d, which is a round-down operation
// see https://github.com/stellar/stellar-core/blob/9af27ef4e20b66f38ab148d52ba7904e74fe502f/src/util/types.cpp#L201
func MulFractionRoundDown(x int64, n int64, d int64) (int64, error) {
	var bn, bd big.Int
	bn.SetInt64(n)
	bd.SetInt64(d)
	var r big.Int

	r.SetInt64(x)
	r.Mul(&r, &bn)
	r.Quo(&r, &bd)

	return toInt64Checked(r)
}

// mulFractionRoundUp sets x = ((x * n) + d - 1) / d, which is a round-up operation
// see https://github.com/stellar/stellar-core/blob/9af27ef4e20b66f38ab148d52ba7904e74fe502f/src/util/types.cpp#L201
func mulFractionRoundUp(x int64, n int64, d int64) (int64, error) {
	var bn, bd big.Int
	bn.SetInt64(n)
	bd.SetInt64(d)
	var one big.Int
	one.SetInt64(1)
	var r big.Int

	r.SetInt64(x)
	r.Mul(&r, &bn)
	r.Add(&r, &bd)
	r.Sub(&r, &one)
	r.Quo(&r, &bd)

	return toInt64Checked(r)
}

func toInt64Checked(x big.Int) (int64, error) {
	if x.IsInt64() {
		return x.Int64(), nil
	}
	return 0, fmt.Errorf("cannot convert big.Int value to int64")
}
