package orderbook

import (
	"math"
	"math/bits"

	"lukechampine.com/uint128"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// There are two different exchanges that can be simulated:
//
// 1. You know how much you can *give* to the pool, and are curious about the
// resulting payout. We call this a "deposit", and you should pass
// tradeTypeDeposit.
//
// 2. You know how much you'd like to *receive* from the pool, and want to know
// how much to deposit to achieve this. We call this an "expectation", and you
// should pass tradeTypeExpectation.
const (
	tradeTypeDeposit     = iota // deposit into pool, what's the payout?
	tradeTypeExpectation = iota // expect payout, what to deposit?
	maxBips              = 10000
)

var (
	errPoolOverflows = errors.New("Liquidity pool overflows from this exchange")
	errBadPoolType   = errors.New("Unsupported liquidity pool: must be ConstantProduct")
	errBadTradeType  = errors.New("Unknown pool exchange type requested")
	errBadAmount     = errors.New("Exchange amount must be positive")
	one128           = uint128.Uint128{1, 0}
	maxBips128       = uint128.From64(maxBips)
)

// makeTrade simulates execution of an exchange with a liquidity pool.
//
// In (1), this returns the amount that would be paid out by the pool (in terms
// of the *other* asset) for depositing `amount` of `asset`.
//
// In (2), this returns the amount of `asset` you'd need to deposit to get
// `amount` of the *other* asset in return.
//
// Refer to https://github.com/stellar/stellar-protocol/blob/master/core/cap-0038.md#pathpaymentstrictsendop-and-pathpaymentstrictreceiveop
// and the calculation functions (below) for details on the exchange algorithm.
//
// Warning: If you pass an asset that is NOT one of the pool reserves, the
// behavior of this function is undefined (for performance).
func makeTrade(
	pool liquidityPool,
	asset int32,
	tradeType int,
	amount xdr.Int64,
) (xdr.Int64, error) {
	details, ok := pool.Body.GetConstantProduct()
	if !ok {
		return 0, errBadPoolType
	}

	if amount <= 0 {
		return 0, errBadAmount
	}

	// determine which asset `amount` corresponds to
	X, Y := details.ReserveA, details.ReserveB
	if pool.assetA != asset {
		X, Y = Y, X
	}

	ok = false
	var result xdr.Int64
	switch tradeType {
	case tradeTypeDeposit:
		result, ok = calculatePoolPayout(X, Y, amount, details.Params.Fee)

	case tradeTypeExpectation:
		result, ok = calculatePoolExpectation(X, Y, amount, details.Params.Fee)

	default:
		return 0, errBadTradeType
	}

	if !ok {
		// the error isn't strictly accurate (e.g. it could be div-by-0), but
		// from the caller's perspective it's true enough
		return 0, errPoolOverflows
	}
	return result, nil
}

func mulWithOverflowCheck(u, v uint128.Uint128) (uint128.Uint128, bool) {
	hi, lo := bits.Mul64(u.Lo, v.Lo)
	p0, p1 := bits.Mul64(u.Hi, v.Lo)
	p2, p3 := bits.Mul64(u.Lo, v.Hi)
	hi, c0 := bits.Add64(hi, p1, 0)
	hi, c1 := bits.Add64(hi, p3, c0)
	overflew := (u.Hi != 0 && v.Hi != 0) || p0 != 0 || p2 != 0 || c1 != 0
	return uint128.Uint128{lo, hi}, !overflew
}

func addWithOverflowCheck(u, v uint128.Uint128) (uint128.Uint128, bool) {
	lo, carry := bits.Add64(u.Lo, v.Lo, 0)
	hi, carry := bits.Add64(u.Hi, v.Hi, carry)
	return uint128.Uint128{lo, hi}, carry == 0
}

// calculatePoolPayout calculates the amount of `reserveB` disbursed from the
// pool for a `received` amount of `reserveA` . From CAP-38:
//
//      y = floor[(1 - F) Yx / (X + x - Fx)]
//
// It returns false if the calculation overflows.
func calculatePoolPayout(reserveA, reserveB, received xdr.Int64, feeBips xdr.Int32) (xdr.Int64, bool) {
	X, Y := uint128.From64(uint64(reserveA)), uint128.From64(uint64(reserveB))
	F, x := uint128.From64(uint64(feeBips)), uint128.From64(uint64(received))

	// would this deposit overflow the reserve?
	// is feeBips within range?
	if received > math.MaxInt64-reserveA {
		return 0, false
	}

	// We do all of the math in bips, so it's all upscaled by this value.
	if feeBips > maxBips {
		return 0, false
	}
	f := maxBips128.Sub(F) // upscaled 1 - F

	// right half: X + (1 - F)x
	denom, ok := mulWithOverflowCheck(X, maxBips128)
	if !ok {
		return 0, false
	}
	xMulf, ok := mulWithOverflowCheck(x, f)
	if !ok {
		return 0, false
	}
	denom, ok = addWithOverflowCheck(denom, xMulf)
	if !ok {
		return 0, false
	}
	if denom.IsZero() { // avoid div-by-zero panic
		return 0, false
	}

	// left half, a: (1 - F) Yx
	numer, ok := mulWithOverflowCheck(Y, x)
	if !ok {
		return 0, false
	}
	numer, ok = mulWithOverflowCheck(numer, f)
	if !ok {
		return 0, false
	}

	// divide & check overflow
	result := numer.Div(denom)

	return xdr.Int64(result.Lo), result.Hi == 0 && result.Lo <= math.MaxInt64
}

// calculatePoolExpectation determines how much of `reserveA` you would need to
// put into a pool to get the `disbursed` amount of `reserveB`.
//
//      x = ceil[Xy / ((Y - y)(1 - F))]
//
// It returns false if the calculation overflows.
func calculatePoolExpectation(
	reserveA, reserveB, disbursed xdr.Int64, feeBips xdr.Int32,
) (xdr.Int64, bool) {
	X, Y := uint128.From64(uint64(reserveA)), uint128.From64(uint64(reserveB))
	F, y := uint128.From64(uint64(feeBips)), uint128.From64(uint64(disbursed))

	// We do all of the math in bips, so it's all upscaled by this value.
	if feeBips > maxBips {
		return 0, false
	}
	f := maxBips128.Sub(F) // upscaled 1 - F

	// right half: (Y - y)(1 - F)
	// sanity check: disbursing shouldn't underflow the reserve
	if disbursed >= reserveB {
		return 0, false
	}
	denom := Y.Sub(y)
	var ok bool
	denom, ok = mulWithOverflowCheck(denom, f)
	if !ok {
		return 0, false
	}

	if denom.IsZero() { // avoid div-by-zero panic
		return 0, false
	}

	// left half: Xy
	numer, ok := mulWithOverflowCheck(X, y)
	if !ok {
		return 0, false
	}
	numer, ok = mulWithOverflowCheck(numer, maxBips128)
	if !ok {
		return 0, false
	}

	result, rem := numer.QuoRem(denom)

	// hacky way to ceil(): if there's a remainder, add 1
	if rem.Cmp64(0) > 0 {
		result, ok = addWithOverflowCheck(result, one128)
		if !ok {
			return 0, false
		}
	}

	return xdr.Int64(result.Lo), result.Hi == 0 && result.Lo <= math.MaxInt64
}

// getOtherAsset returns the other asset in the liquidity pool. Note that
// doesn't check to make sure the passed in `asset` is actually part of the
// pool; behavior in that case is undefined.
func getOtherAsset(asset int32, pool liquidityPool) int32 {
	if pool.assetA == asset {
		return pool.assetB
	}
	return pool.assetA
}
