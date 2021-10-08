package orderbook

import (
	"math"
	"math/big"

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
)

var (
	errPoolOverflows = errors.New("Liquidity pool overflows from this exchange")
	errBadPoolType   = errors.New("Unsupported liquidity pool: must be ConstantProduct")
	errBadTradeType  = errors.New("Unknown pool exchange type requested")
	errBadAmount     = errors.New("Exchange amount must be positive")
)

// makeTrade simulates execution of an exchange with a liquidity pool.
//
// In (1), this returns the amount that would be paid out by the pool (in terms
// of the *other* asset) for depositing `amount` of `asset`.
//
// In (2), this returns the amount of `asset` necessary to give to the pool in
// order to get `amount` of the other asset in return.
//
// Refer to https://github.com/stellar/stellar-protocol/blob/master/core/cap-0038.md#pathpaymentstrictsendop-and-pathpaymentstrictreceiveop
// and the calculation functions (below) for details on the exchange algorithm.
//
// Warning: If you pass an asset that is NOT one of the pool reserves, the
// behavior of this function is undefined (for performance).
func makeTrade(
	pool xdr.LiquidityPoolEntry,
	asset xdr.Asset,
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
	if !details.Params.AssetA.Equals(asset) {
		X, Y = details.ReserveB, details.ReserveA
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

// calculatePoolPayout calculates the amount disbursed from the pool for an
// amount received. From CAP-38:
//
//      y = floor[(1 - F) Yx / (X + x - Fx)]
//
// It returns false if the calculation overflows.
func calculatePoolPayout(reserveA, reserveB, received xdr.Int64, feeBips xdr.Int32) (xdr.Int64, bool) {
	X, Y := big.NewInt(int64(reserveA)), big.NewInt(int64(reserveB))
	F, x := big.NewInt(int64(feeBips)), big.NewInt(int64(received))

	// would this deposit overflow the reserve?
	if math.MaxInt64-received < reserveA {
		return 0, false
	}

	// We do all of the math in bips, so it's all upscaled by this value.
	maxBips := big.NewInt(10000)
	f := new(big.Int).Sub(maxBips, F) // upscaled 1 - F

	// right half: X + (1 - F)x
	denom := X.Mul(X, maxBips).Add(X, new(big.Int).Mul(x, f))
	if denom.Cmp(big.NewInt(0)) == 0 { // avoid div-by-zero panic
		return 0, false
	}

	// left half, a: (1 - F) Yx
	numer := Y.Mul(Y, x).Mul(Y, f)

	// divide & check overflow
	result := numer.Div(numer, denom)

	return xdr.Int64(result.Int64()), result.IsInt64()
}

// calculatePoolExpectation determines how much you would need to put into a
// pool to get a certain amount disbursed.
//
//      x = ceil[Xy / (Y - y) / (1 - F)]
//
// It returns false if the calculation overflows.
func calculatePoolExpectation(
	reserveA, reserveB, disbursed xdr.Int64, feeBips xdr.Int32,
) (xdr.Int64, bool) {
	X, Y := big.NewInt(int64(reserveA)), big.NewInt(int64(reserveB))
	F, y := big.NewInt(int64(feeBips)), big.NewInt(int64(disbursed))

	// sanity check: disbursing shouldn't underflow the reserve
	if reserveA-disbursed <= 0 {
		return 0, false
	}

	// We do all of the math in bips, so it's all upscaled by this value.
	maxBips := big.NewInt(10000)
	f := new(big.Int).Sub(maxBips, F) // upscaled 1 - F

	denom := Y.Sub(Y, y).Mul(Y, f)     // right half:
	if denom.Cmp(big.NewInt(0)) == 0 { // avoid div-by-zero panic
		return 0, false
	}

	numer := X.Mul(X, y).Mul(X, maxBips)

	result, rem := new(big.Int), new(big.Int)
	result.DivMod(numer, denom, rem)

	// hacky way to ceil(): if there's a remainder, add 1
	if rem.Cmp(big.NewInt(0)) > 0 {
		result.Add(result, big.NewInt(1))
	}

	return xdr.Int64(result.Int64()), result.IsInt64()
}

// getOtherAsset returns the other asset in the liquidity pool. Note that
// doesn't check to make sure the passed in `asset` is actually part of the
// pool; behavior in that case is undefined.
func getOtherAsset(asset xdr.Asset, pool xdr.LiquidityPoolEntry) xdr.Asset {
	cp := pool.Body.MustConstantProduct()
	if cp.Params.AssetA.Equals(asset) {
		return cp.Params.AssetB
	}
	return cp.Params.AssetA
}
