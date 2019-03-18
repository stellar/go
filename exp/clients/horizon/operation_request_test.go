package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationRequestBuildUrl(t *testing.T) {
	op := OperationRequest{}
	endpoint, err := op.BuildUrl()

	// It should return valid all operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "operations", endpoint)

	op = OperationRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err = op.BuildUrl()

	// It should return valid account operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/operations", endpoint)

	op = OperationRequest{ForLedger: 123}
	endpoint, err = op.BuildUrl()

	// It should return valid ledger operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123/operations", endpoint)

	op = OperationRequest{forOperationId: "123"}
	endpoint, err = op.BuildUrl()

	// It should return valid operation operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "operations/123", endpoint)

	op = OperationRequest{ForTransaction: "123"}
	endpoint, err = op.BuildUrl()

	// It should return valid transaction operations endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions/123/operations", endpoint)

	op = OperationRequest{ForLedger: 123, forOperationId: "789"}
	endpoint, err = op.BuildUrl()

	// error case: too many parameters for building any operation endpoint
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid request. Too many parameters")
	}

	op = OperationRequest{Cursor: "123456", Limit: 30, Order: OrderAsc}
	endpoint, err = op.BuildUrl()
	// It should return valid all operations endpoint with query params and no errors
	require.NoError(t, err)
	assert.Equal(t, "operations?cursor=123456&limit=30&order=asc", endpoint)

}
