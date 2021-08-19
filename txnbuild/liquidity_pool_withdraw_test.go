package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiquidityPoolWithdrawRoundTrip(t *testing.T) {
	assetA := NativeAsset{}
	assetB := CreditAsset{
		Code:   "EUR",
		Issuer: "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
	}
	poolId, err := NewLiquidityPoolId(assetA, assetB)
	require.NoError(t, err)

	lpd := &LiquidityPoolWithdraw{
		SourceAccount:   "GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H",
		LiquidityPoolID: poolId,
		Amount:          "0.1000000",
		MinAmountA:      "0.1000000",
		MinAmountB:      "0.2000000",
	}

	testOperationsMarshallingRoundtrip(t, []Operation{lpd}, false)

	// with muxed accounts
	lpd = &LiquidityPoolWithdraw{
		SourceAccount:   "MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK",
		LiquidityPoolID: poolId,
		Amount:          "0.1000000",
		MinAmountA:      "0.1000000",
		MinAmountB:      "0.2000000",
	}

	testOperationsMarshallingRoundtrip(t, []Operation{lpd}, true)
}
