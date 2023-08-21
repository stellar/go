package history

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestClaimableBalanceLoader(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	session := tt.HorizonSession()

	var ids []string
	for i := 0; i < 100; i++ {
		balanceID := xdr.ClaimableBalanceId{
			Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
			V0:   &xdr.Hash{byte(i)},
		}
		id, err := xdr.MarshalHex(balanceID)
		tt.Assert.NoError(err)
		ids = append(ids, id)
	}

	loader := NewClaimableBalanceLoader()
	var futures []FutureClaimableBalanceID
	for _, id := range ids {
		future := loader.GetFuture(id)
		futures = append(futures, future)
		assert.Panics(t, func() {
			loader.getNow(id)
		})
		assert.Panics(t, func() {
			future.Value()
		})
	}

	assert.NoError(t, loader.Exec(context.Background(), session))
	assert.Panics(t, func() {
		loader.GetFuture("not-present")
	})

	q := &Q{session}
	for i, id := range ids {
		future := futures[i]
		internalID := loader.getNow(id)
		val, err := future.Value()
		assert.NoError(t, err)
		assert.Equal(t, internalID, val)
		cb, err := q.ClaimableBalanceByID(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, cb.BalanceID, id)
		assert.Equal(t, cb.InternalID, internalID)
	}
}
