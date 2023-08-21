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

func TestAssetLoader(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	session := tt.HorizonSession()

	var keys []AssetKey
	for i := 0; i < 100; i++ {
		var key AssetKey
		if i == 0 {
			key.Type = "native"
		} else if i%2 == 0 {
			key.Type = "credit_alphanum4"
			key.Code = fmt.Sprintf("ab%d", i)
			key.Issuer = keypair.MustRandom().Address()
		} else {
			key.Type = "credit_alphanum12"
			key.Code = fmt.Sprintf("abcdef%d", i)
			key.Issuer = keypair.MustRandom().Address()
		}
		keys = append(keys, key)
	}

	loader := NewAssetLoader()
	var futures []FutureAssetID
	for _, key := range keys {
		future := loader.GetFuture(key)
		futures = append(futures, future)
		assert.Panics(t, func() {
			loader.getNow(key)
		})
		assert.Panics(t, func() {
			future.Value()
		})
	}

	assert.NoError(t, loader.Exec(context.Background(), session))
	assert.Panics(t, func() {
		loader.GetFuture(AssetKey{Type: "invalid"})
	})

	q := &Q{session}
	for i, key := range keys {
		future := futures[i]
		internalID := loader.getNow(key)
		val, err := future.Value()
		assert.NoError(t, err)
		assert.Equal(t, internalID, val)
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
}
