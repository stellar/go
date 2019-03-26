package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionRequestBuildUrl(t *testing.T) {
	tr := TransactionRequest{}
	endpoint, err := tr.BuildUrl()

	// It should return valid all transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions", endpoint)

	tr = TransactionRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err = tr.BuildUrl()

	// It should return valid account transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/transactions", endpoint)

	tr = TransactionRequest{ForLedger: 123}
	endpoint, err = tr.BuildUrl()

	// It should return valid ledger transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "ledgers/123/transactions", endpoint)

	tr = TransactionRequest{forTransactionHash: "123"}
	endpoint, err = tr.BuildUrl()

	// It should return valid operation transactions endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions/123", endpoint)

	tr = TransactionRequest{ForLedger: 123, forTransactionHash: "789"}
	endpoint, err = tr.BuildUrl()

	// error case: too many parameters for building any operation endpoint
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid request. Too many parameters")
	}

	tr = TransactionRequest{Cursor: "123456", Limit: 30, Order: OrderAsc, IncludeFailed: true}
	endpoint, err = tr.BuildUrl()
	// It should return valid all transactions endpoint with query params and no errors
	require.NoError(t, err)
	assert.Equal(t, "transactions?cursor=123456&include_failed=true&limit=30&order=asc", endpoint)

}
