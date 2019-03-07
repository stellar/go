package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectRequestBuildUrl(t *testing.T) {
	er := EffectRequest{}
	endpoint, err := er.BuildUrl()

	// It should return valid all effects endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "effects", endpoint)

	er = EffectRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err = er.BuildUrl()

	// It should return valid account effects endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/effects", endpoint)

	er = EffectRequest{ForLedger: "123"}
	endpoint, err = er.BuildUrl()

	// It should return valid ledger effects endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123/effects", endpoint)

	er = EffectRequest{ForOperation: "123"}
	endpoint, err = er.BuildUrl()

	// It should return valid operation effects endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "operations/123/effects", endpoint)

	er = EffectRequest{ForTransaction: "123"}
	endpoint, err = er.BuildUrl()

	// It should return valid transaction effects endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions/123/effects", endpoint)

	er = EffectRequest{ForLedger: "123", ForOperation: "789"}
	endpoint, err = er.BuildUrl()

	// error case: too many parameters for building any effect endpoint
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid request. Too many parameters")
	}

	er = EffectRequest{Cursor: "123456", Limit: 30, Order: OrderAsc}
	endpoint, err = er.BuildUrl()
	// It should return valid all effects endpoint with query params and no errors
	require.NoError(t, err)
	assert.Equal(t, "effects?cursor=123456&limit=30&order=asc", endpoint)

}
