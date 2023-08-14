package history

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestAccountLoader(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	session := tt.HorizonSession()

	var addresses []string
	for i := 0; i < 100; i++ {
		addresses = append(addresses, keypair.MustRandom().Address())
	}

	loader := NewAccountLoader()
	var futures []FutureAccountID
	for _, address := range addresses {
		future := loader.GetFuture(address)
		futures = append(futures, future)
		assert.Panics(t, func() {
			loader.GetNow(address)
		})
		assert.Panics(t, func() {
			future.Value()
		})
	}

	assert.NoError(t, loader.Exec(context.Background(), session))
	assert.Panics(t, func() {
		loader.GetFuture(keypair.MustRandom().Address())
	})

	q := &Q{session}
	for i, address := range addresses {
		future := futures[i]
		id := loader.GetNow(address)
		val, err := future.Value()
		assert.NoError(t, err)
		assert.Equal(t, id, val)
		var account Account
		assert.NoError(t, q.AccountByAddress(context.Background(), &account, address))
		assert.Equal(t, account.ID, id)
		assert.Equal(t, account.Address, address)
	}
}
