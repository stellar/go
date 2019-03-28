package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the TradeAggregationRequest struct.
func (ta TradeAggregationRequest) BuildUrl() (endpoint string, err error) {
	endpoint = "trade_aggregations"
	// add the parameters for trade aggregations endpoint
	paramMap := make(map[string]string)
	paramMap["start_time"] = ta.StartTime
	paramMap["end_time"] = ta.EndTime
	paramMap["resolution"] = ta.Resolution
	paramMap["offset"] = ta.Offset
	paramMap["base_asset_type"] = string(ta.BaseAssetType)
	paramMap["base_asset_code"] = ta.BaseAssetCode
	paramMap["base_asset_issuer"] = ta.BaseAssetIssuer
	paramMap["counter_asset_type"] = string(ta.CounterAssetType)
	paramMap["counter_asset_code"] = ta.CounterAssetCode
	paramMap["counter_asset_issuer"] = ta.CounterAssetIssuer

	queryParams := addQueryParams(paramMap, limit(ta.Limit), ta.Order)
	if queryParams != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, queryParams)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
