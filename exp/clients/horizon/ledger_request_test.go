package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLedgerRequestBuildUrl(t *testing.T) {
	lr := LedgerRequest{}
	endpoint, err := lr.BuildUrl()

	// It should return valid all ledgers endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers", endpoint)

	lr = LedgerRequest{forSequence: 123}
	endpoint, err = lr.BuildUrl()

	// It should return valid ledger detail endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123", endpoint)

	lr = LedgerRequest{forSequence: 123, Cursor: "now", Order: OrderDesc}
	endpoint, err = lr.BuildUrl()

	// It should return valid ledger detail endpoint, with no cursor or order
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123", endpoint)

	lr = LedgerRequest{Cursor: "now", Order: OrderDesc}
	endpoint, err = lr.BuildUrl()

	// It should return valid ledgers endpoint, with cursor and order
	require.NoError(t, err)
	assert.Equal(t, "ledgers?cursor=now&order=desc", endpoint)
}
