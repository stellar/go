package horizonclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/stellar/go/protocols/horizon/operations"
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

// SetPaymentsEndpoint is a helper function that sets the `endpoint` for OperationRequests to `payments`
func (op *OperationRequest) SetPaymentsEndpoint() *OperationRequest {
	return op.setEndpoint("payments")
}

// SetOperationsEndpoint is a helper function that sets the `endpoint` for OperationRequests to `operations`
func (op *OperationRequest) SetOperationsEndpoint() *OperationRequest {
	return op.setEndpoint("operations")
}

// OperationHandler is a function that is called when a new operation is received
type OperationHandler func(operations.Operation)

// StreamOperations streams stellar operations. It can be used to stream all operations or operations
// for and account. Use context.WithCancel to stop streaming or context.Background() if you want to
// stream indefinitely. OperationHandler is a user-supplied function that is executed for each streamed
//  operation received.
func (op OperationRequest) StreamOperations(ctx context.Context, client *Client, handler OperationHandler) error {
	endpoint, err := op.BuildUrl()
	if err != nil {
		return errors.Wrap(err, "Unable to build endpoint for operation request")
	}

	url := fmt.Sprintf("%s%s", client.getHorizonURL(), endpoint)
	return client.stream(ctx, url, func(data []byte) error {
		var baseRecord operations.Base

		if err = json.Unmarshal(data, &baseRecord); err != nil {
			return errors.Wrap(err, "Error unmarshaling data for operation request")
		}

		ops, err := operations.UnmarshalOperation(baseRecord.GetType(), data)
		if err != nil {
			return errors.Wrap(err, "Unmarshaling to the correct operation type")
		}

		handler(ops)
		return nil
	})
}
