package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestAccountQueries(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	// Test Accounts()
	acs := []Account{}
	err := q.Accounts().Select(&acs)

	if tt.Assert.NoError(err) {
		tt.Assert.Len(acs, 4)
	}
}

func TestIsAuthRequired(t *testing.T) {
	tt := assert.New(t)

	account := AccountEntry{Flags: 1}
	tt.True(account.IsAuthRequired())

	account = AccountEntry{Flags: 0}
	tt.False(account.IsAuthRequired())
}

func TestIsAuthRevocable(t *testing.T) {
	tt := assert.New(t)

	account := AccountEntry{Flags: 2}
	tt.True(account.IsAuthRevocable())

	account = AccountEntry{Flags: 1}
	tt.False(account.IsAuthRevocable())
}
func TestIsAuthImmutable(t *testing.T) {
	tt := assert.New(t)

	account := AccountEntry{Flags: 4}
	tt.True(account.IsAuthImmutable())

	account = AccountEntry{Flags: 0}
	tt.False(account.IsAuthImmutable())
}
