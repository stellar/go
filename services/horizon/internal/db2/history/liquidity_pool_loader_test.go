package history

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestLiquidityPoolLoader(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	session := tt.HorizonSession()

	var ids []string
	for i := 0; i < 100; i++ {
		poolID := xdr.PoolId{byte(i)}
		id, err := xdr.MarshalHex(poolID)
		tt.Assert.NoError(err)
		ids = append(ids, id)
	}

	loader := NewLiquidityPoolLoader()
	var futures []FutureLiquidityPoolID
	for _, id := range ids {
		future := loader.GetFuture(id)
		futures = append(futures, future)
		assert.Panics(t, func() {
			loader.GetNow(id)
		})
		assert.Panics(t, func() {
			future.Value()
		})
		duplicateFuture := loader.GetFuture(id)
		assert.Equal(t, future, duplicateFuture)
	}

	assert.NoError(t, loader.Exec(context.Background(), session))
	assert.Panics(t, func() {
		loader.GetFuture("not-present")
	})

	q := &Q{session}
	for i, id := range ids {
		future := futures[i]
		internalID := loader.GetNow(id)
		val, err := future.Value()
		assert.NoError(t, err)
		assert.Equal(t, internalID, val)
		lp, err := q.LiquidityPoolByID(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, lp.PoolID, id)
		assert.Equal(t, lp.InternalID, internalID)
	}
}
