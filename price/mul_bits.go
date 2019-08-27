// This implementation makes use of bits.Mul64, bits.Div64, and bits.Add64
// which are only supported in go 1.12 or higher
//
// +build go1.12

package price

import (
	"errors"
	"math"
	"math/bits"
)

// MulFractionRoundDown sets x = (x * n) / d, which is a round-down operation
// see https://github.com/stellar/stellar-core/blob/9af27ef4e20b66f38ab148d52ba7904e74fe502f/src/util/types.cpp#L201
func MulFractionRoundDown(x int64, n int64, d int64) (int64, error) {
	if d == 0 {
		return 0, errors.New("division by 0")
	}

	hi, lo := bits.Mul64(uint64(x), uint64(n))

	denominator := uint64(d)
	if denominator <= hi {
		return 0, errors.New("overflow")
	}
	q, _ := bits.Div64(hi, lo, denominator)
	if q > math.MaxInt64 {
		return 0, errors.New("overflow")
	}

	return int64(q), nil
}

// mulFractionRoundUp sets x = ((x * n) + d - 1) / d, which is a round-up operation
// see https://github.com/stellar/stellar-core/blob/9af27ef4e20b66f38ab148d52ba7904e74fe502f/src/util/types.cpp#L201
func mulFractionRoundUp(x int64, n int64, d int64) (int64, error) {
	if d == 0 {
		return 0, errors.New("division by 0")
	}

	hi, lo := bits.Mul64(uint64(x), uint64(n))
	lo, carry := bits.Add64(lo, uint64(d-1), 0)
	hi += carry

	denominator := uint64(d)
	if denominator <= hi {
		return 0, errors.New("overflow")
	}
	q, _ := bits.Div64(hi, lo, denominator)
	if q > math.MaxInt64 {
		return 0, errors.New("overflow")
	}

	return int64(q), nil
}
