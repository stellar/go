package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the TradeRequest struct.
// If no data is set, it defaults to the build the URL for all trades
func (tr TradeRequest) BuildUrl() (endpoint string, err error) {
	nParams := countParams(tr.ForAccount, tr.ForOfferID)

	if nParams > 1 {
		return endpoint, errors.New("Invalid request. Too many parameters")
	}

	endpoint = "trades"
	if tr.ForAccount != "" {
		endpoint = fmt.Sprintf("accounts/%s/trades", tr.ForAccount)
	}

	// Note[Peter - 28/03/2019]: querying an "all trades" endpoint that has the query parameter
	// for offer_id is the same as querying the url for trades of a particular offer. The results
	// returned will be the same. So, I am opting to build the endpoint for trades per offer when
	// `ForOfferID` is set
	if tr.ForOfferID != "" {
		endpoint = fmt.Sprintf("offers/%s/trades", tr.ForOfferID)
	}

	var queryParams string

	if endpoint != "trades" {
		queryParams = addQueryParams(cursor(tr.Cursor), limit(tr.Limit), tr.Order)
	} else {
		// add the parameters for all trades endpoint
		paramMap := make(map[string]string)
		paramMap["base_asset_type"] = string(tr.BaseAssetType)
		paramMap["base_asset_code"] = tr.BaseAssetCode
		paramMap["base_asset_issuer"] = tr.BaseAssetIssuer
		paramMap["counter_asset_type"] = string(tr.CounterAssetType)
		paramMap["counter_asset_code"] = tr.CounterAssetCode
		paramMap["counter_asset_issuer"] = tr.CounterAssetIssuer
		paramMap["offer_id"] = tr.ForOfferID

		queryParams = addQueryParams(paramMap, cursor(tr.Cursor), limit(tr.Limit), tr.Order)
	}

	if queryParams != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, queryParams)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
