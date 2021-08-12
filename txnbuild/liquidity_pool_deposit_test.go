package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiquidityPoolDepositRoundTrip(t *testing.T) {
	assetA := NativeAsset{}
	assetB := CreditAsset{
		Code:   "EUR",
		Issuer: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
	}
	poolId, err := NewLiquidityPoolId(assetA, assetB)
	require.NoError(t, err)

	lpd := &LiquidityPoolDeposit{
		SourceAccount:   "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		LiquidityPoolID: poolId,
		MaxAmountA:      "0.1000000",
		MaxAmountB:      "0.2000000",
		MinPrice:        price{n: 1, d: 2, s: "0.5"},
		MaxPrice:        price{n: 1, d: 2, s: "0.5"},
	}

	testOperationsMarshallingRoundtrip(t, []Operation{lpd}, false)

	// with muxed accounts
	lpd = &LiquidityPoolDeposit{
		SourceAccount:   "MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK",
		LiquidityPoolID: poolId,
		MaxAmountA:      "0.1000000",
		MaxAmountB:      "0.2000000",
		MinPrice:        price{n: 1, d: 2, s: "0.5"},
		MaxPrice:        price{n: 1, d: 2, s: "0.5"},
	}

	testOperationsMarshallingRoundtrip(t, []Operation{lpd}, true)
}
