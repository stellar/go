package txnbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLiquidityPoolId(t *testing.T) {
	a := NativeAsset{}
	b := CreditAsset{Code: "ABC", Issuer: "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"}

	id, err := NewLiquidityPoolId(a, b)
	if assert.NoError(t, err) {
		assert.Equal(t, id, LiquidityPoolId([32]byte{
			0x99, 0xd6, 0xb8, 0x3f, 0x51, 0xf3, 0x58, 0x48,
			0x0d, 0x79, 0xa7, 0xc7, 0xc8, 0xb5, 0xf6, 0xd5,
			0x3e, 0x7f, 0x7f, 0xcf, 0x51, 0x49, 0x31, 0x64,
			0xff, 0xb2, 0xed, 0xc2, 0xfd, 0xd6, 0xfc, 0x9e,
		}), "pool id should match")
	}

	// Wrong asset id order should fail. If users mess this up, and we were to
	// silently fix it they could set the wrong MaxAmounts when depositing.
	_, err = NewLiquidityPoolId(b, a)
	assert.EqualError(t, err, "AssetA must be <= AssetB")
}
