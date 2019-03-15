package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the OperationRequest struct.
// If no data is set, it defaults to the build the URL for all operations
func (op OperationRequest) BuildUrl() (endpoint string, err error) {

	nParams := countParams(op.ForAccount, op.ForLedger, op.forOperationId, op.ForTransaction)

	if nParams > 1 {
		err = errors.New("Invalid request. Too many parameters")
	}

	if err != nil {
		return endpoint, err
	}

	endpoint = "operations"

	if op.ForAccount != "" {
		endpoint = fmt.Sprintf(
			"accounts/%s/operations",
			op.ForAccount,
		)
	}

	if op.ForLedger > 0 {
		endpoint = fmt.Sprintf(
			"ledgers/%d/operations",
			op.ForLedger,
		)
	}

	if op.forOperationId > 0 {
		endpoint = fmt.Sprintf(
			"operations/%d",
			op.forOperationId,
		)
	}

	if op.ForTransaction != "" {
		endpoint = fmt.Sprintf(
			"transactions/%s/operations",
			op.ForTransaction,
		)
	}

	queryParams := addQueryParams(op.Cursor, op.Limit, op.Order, op.IncludeFailed)
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
