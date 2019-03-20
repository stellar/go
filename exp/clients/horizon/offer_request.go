package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the OfferRequest struct.
func (or OfferRequest) BuildUrl() (endpoint string, err error) {
	endpoint = fmt.Sprintf(
		"accounts/%s/offers",
		or.ForAccount,
	)

	queryParams := addQueryParams(cursor(or.Cursor), limit(or.Limit), or.Order)
	if queryParams != "" {
		endpoint = fmt.Sprintf(
			"%s?%s",
			endpoint,
			queryParams,
		)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
