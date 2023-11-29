package history

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestAssetKeyToString(t *testing.T) {
	num4key := AssetKey{
		Type:   "credit_alphanum4",
		Code:   "USD",
		Issuer: "A1B2C3",
	}

	num12key := AssetKey{
		Type:   "credit_alphanum12",
		Code:   "USDABC",
		Issuer: "A1B2C3",
	}

	nativekey := AssetKey{
		Type: "native",
	}

	assert.Equal(t, num4key.String(), "credit_alphanum4/USD/A1B2C3")
	assert.Equal(t, num12key.String(), "credit_alphanum12/USDABC/A1B2C3")
	assert.Equal(t, nativekey.String(), "native")
}

func TestAssetLoader(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	session := tt.HorizonSession()

	var keys []AssetKey
	for i := 0; i < 100; i++ {
		var key AssetKey
		if i == 0 {
			key = AssetKeyFromXDR(xdr.Asset{Type: xdr.AssetTypeAssetTypeNative})
		} else if i%2 == 0 {
			code := [4]byte{0, 0, 0, 0}
			copy(code[:], fmt.Sprintf("ab%d", i))
			key = AssetKeyFromXDR(xdr.Asset{
				Type: xdr.AssetTypeAssetTypeCreditAlphanum4,
				AlphaNum4: &xdr.AlphaNum4{
					AssetCode: code,
					Issuer:    xdr.MustAddress(keypair.MustRandom().Address())}})
		} else {
			code := [12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
			copy(code[:], fmt.Sprintf("abcdef%d", i))
			key = AssetKeyFromXDR(xdr.Asset{
				Type: xdr.AssetTypeAssetTypeCreditAlphanum12,
				AlphaNum12: &xdr.AlphaNum12{
					AssetCode: code,
					Issuer:    xdr.MustAddress(keypair.MustRandom().Address())}})

		}
		keys = append(keys, key)
	}

	loader := NewAssetLoader()
	for _, key := range keys {
		future := loader.GetFuture(key)
		_, err := future.Value()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `invalid asset loader state,`)
		duplicateFuture := loader.GetFuture(key)
		assert.Equal(t, future, duplicateFuture)
	}

	assert.NoError(t, loader.Exec(context.Background(), session))
	assert.Panics(t, func() {
		loader.GetFuture(AssetKey{Type: "invalid"})
	})

	q := &Q{session}
	for _, key := range keys {
		internalID, err := loader.GetNow(key)
		assert.NoError(t, err)
		var assetXDR xdr.Asset
		if key.Type == "native" {
			assetXDR = xdr.MustNewNativeAsset()
		} else {
			assetXDR = xdr.MustNewCreditAsset(key.Code, key.Issuer)
		}
		assetID, err := q.GetAssetID(context.Background(), assetXDR)
		assert.NoError(t, err)
		assert.Equal(t, assetID, internalID)
	}

	_, err := loader.GetNow(AssetKey{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `was not found`)
}
