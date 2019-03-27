package horizonclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderBookRequestBuildUrl(t *testing.T) {
	obr := OrderBookRequest{}
	endpoint, err := obr.BuildUrl()

	// It should return no errors and orderbook endpoint
	// Horizon will return an error though because there are no parameters
	require.NoError(t, err)
	assert.Equal(t, "order_book", endpoint)

	obr = OrderBookRequest{SellingAssetType: AssetTypeNative, BuyingAssetType: AssetTypeNative}
	endpoint, err = obr.BuildUrl()

	// It should return valid assets endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "order_book?buying_asset_type=native&selling_asset_type=native", endpoint)

	obr = OrderBookRequest{SellingAssetType: AssetTypeNative, BuyingAssetType: AssetType4, BuyingAssetCode: "ABC", BuyingAssetIssuer: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err = obr.BuildUrl()

	// It should return valid assets endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "order_book?buying_asset_code=ABC&buying_asset_issuer=GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU&buying_asset_type=credit_alphanum4&selling_asset_type=native", endpoint)
}
