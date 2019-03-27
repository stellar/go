package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the OrderBookRequest struct.
func (obr OrderBookRequest) BuildUrl() (endpoint string, err error) {
	endpoint = "order_book"

	// add the parameters to a map here so it is easier for addQueryParams to populate the parameter list
	// We can't use assetCode and assetIssuer types here because the paremeter names are different
	paramMap := make(map[string]string)
	paramMap["selling_asset_type"] = string(obr.SellingAssetType)
	paramMap["selling_asset_code"] = obr.SellingAssetCode
	paramMap["selling_asset_issuer"] = obr.SellingAssetIssuer
	paramMap["buying_asset_type"] = string(obr.BuyingAssetType)
	paramMap["buying_asset_code"] = obr.BuyingAssetCode
	paramMap["buying_asset_issuer"] = obr.BuyingAssetIssuer

	queryParams := addQueryParams(paramMap, limit(obr.Limit))
	if queryParams != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, queryParams)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
