package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the LedgerRequest struct.
// If no data is set, it defaults to the build the URL for all ledgers
func (lr LedgerRequest) BuildUrl() (endpoint string, err error) {
	endpoint = "ledgers"

	if lr.forSequence != 0 {
		endpoint = fmt.Sprintf(
			"%s/%d",
			endpoint,
			lr.forSequence,
		)
	} else {
		queryParams := addQueryParams(cursor(lr.Cursor), limit(lr.Limit), lr.Order)
		if queryParams != "" {
			endpoint = fmt.Sprintf(
				"%s?%s",
				endpoint,
				queryParams,
			)
		}
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
