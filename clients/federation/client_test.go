package federation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupByName(t *testing.T) {
	// HACK: until we improve our mocking scenario, this is just a smoke test
	// against a known federation server.  When/if it breaks, please write this
	// test correctly.

	r, err := DefaultPublicNetClient.LookupByAddress("nullstyle*codetip.io")
	assert.NoError(t, err)
	assert.Equal(t, "GASTNVNLHVR3NFO3QACMHCJT3JUSIV4NBXDHDO4VTPDTNN65W3B2766C", r.AccountID)
}

func TestLookupByID(t *testing.T) {
	// HACK: until we improve our mocking scenario, this is just a smoke test.
	// When/if it breaks, please write this test correctly.  That, or curse
	// scott's name aloud.

	// an account without a homedomain set fails
	_, err := DefaultPublicNetClient.LookupByAccountID("GASTNVNLHVR3NFO3QACMHCJT3JUSIV4NBXDHDO4VTPDTNN65W3B2766C")
	assert.Error(t, err)
	assert.Equal(t, "homedomain not set", err.Error())
}
