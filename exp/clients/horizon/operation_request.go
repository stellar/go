package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the OperationRequest struct.
// If no data is set, it defaults to the build the URL for all operations or all payments; depending on thevalue of `op.endpoint`
func (op OperationRequest) BuildUrl() (endpoint string, err error) {
	nParams := countParams(op.ForAccount, op.ForLedger, op.forOperationId, op.ForTransaction)

	if nParams > 1 {
		return endpoint, errors.New("Invalid request. Too many parameters")
	}

	if op.endpoint == "" {
		return endpoint, errors.New("Internal error, endpoint not set")
	}

	endpoint = op.endpoint
	if op.ForAccount != "" {
		endpoint = fmt.Sprintf("accounts/%s/%s", op.ForAccount, op.endpoint)
	}
	if op.ForLedger > 0 {
		endpoint = fmt.Sprintf("ledgers/%d/%s", op.ForLedger, op.endpoint)
	}
	if op.forOperationId != "" {
		endpoint = fmt.Sprintf("operations/%s", op.forOperationId)
	}
	if op.ForTransaction != "" {
		endpoint = fmt.Sprintf("transactions/%s/%s", op.ForTransaction, op.endpoint)
	}

	queryParams := addQueryParams(cursor(op.Cursor), limit(op.Limit), op.Order,
		includeFailed(op.IncludeFailed))
	if queryParams != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, queryParams)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}

// setEndpoint sets the endpoint for the OperationRequest
func (op *OperationRequest) setEndpoint(endpoint string) *OperationRequest {
	if endpoint == "payments" {
		op.endpoint = endpoint
	} else {
		// default to operations
		op.endpoint = "operations"
	}

	return op
}
