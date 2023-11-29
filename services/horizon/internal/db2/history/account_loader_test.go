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
	for _, address := range addresses {
		future := loader.GetFuture(address)
		_, err := future.Value()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `invalid account loader state,`)
		duplicateFuture := loader.GetFuture(address)
		assert.Equal(t, future, duplicateFuture)
	}

	assert.NoError(t, loader.Exec(context.Background(), session))
	assert.Panics(t, func() {
		loader.GetFuture(keypair.MustRandom().Address())
	})

	q := &Q{session}
	for _, address := range addresses {
		internalId, err := loader.GetNow(address)
		assert.NoError(t, err)
		var account Account
		assert.NoError(t, q.AccountByAddress(context.Background(), &account, address))
		assert.Equal(t, account.ID, internalId)
		assert.Equal(t, account.Address, address)
	}

	_, err := loader.GetNow("not present")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `was not found`)
}
