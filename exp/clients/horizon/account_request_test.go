package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRequestBuildUrl(t *testing.T) {
	ar := AccountRequest{}
	endpoint, err := ar.BuildUrl()

	// error case: No parameters
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid request. No parameters")
	}

	ar.DataKey = "test"
	endpoint, err = ar.BuildUrl()

	// error case: few parameters for building account data endpoint
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid request. Too few parameters")
	}

	ar.DataKey = ""
	ar.AccountId = "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"
	endpoint, err = ar.BuildUrl()

	// It should return valid account details endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU", endpoint)

	ar.DataKey = "test"
	ar.AccountId = "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"
	endpoint, err = ar.BuildUrl()

	// It should return valid account data endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/data/test", endpoint)
}
