package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfferRequestBuildUrl(t *testing.T) {

	or := OfferRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err := or.BuildUrl()

	// It should return valid offers endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/offers", endpoint)

	or = OfferRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU", RequestParams: RequestParams{Cursor: "now", Order: OrderDesc}}
	endpoint, err = or.BuildUrl()

	// It should return valid offers endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/offers?cursor=now&order=desc", endpoint)
}
