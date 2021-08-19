package txnbuild

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssetsSorting(t *testing.T) {
	// Native is always first
	a := NativeAsset{}

	// Type is Alphanum4
	b := CreditAsset{Code: "BCDE", Issuer: "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL5"}

	// Type is Alphanum12
	c := CreditAsset{Code: "ABCD1", Issuer: "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"}

	// Code is >
	d := CreditAsset{Code: "ABCD2", Issuer: "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"}

	// Issuer is >
	e := CreditAsset{Code: "ABCD2", Issuer: "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL4"}

	expected := Assets([]Asset{a, b, c, d, e})

	// Sanity check it doesn't just panic or mess stuff up
	assets := Assets([]Asset{a, b, c, d, e})
	sort.Sort(assets)
	assert.Equal(t, expected, assets)

	// Reverse it and check it still sorts to the same
	assets = Assets([]Asset{e, d, c, b, a})
	sort.Sort(assets)
	assert.Equal(t, expected, assets)
}
