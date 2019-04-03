package horizonclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	hProtocol "github.com/stellar/go/protocols/horizon"
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

// Stream streams incoming ledgers. Use context.WithCancel to stop streaming or
// context.Background() if you want to stream indefinitely.
func (lr LedgerRequest) Stream(ctx context.Context, client *Client,
	handler func(interface{})) (err error) {
	endpoint, err := lr.BuildUrl()
	if err != nil {
		return errors.Wrap(err, "Unable to build endpoint")
	}

	url := fmt.Sprintf("%s%s", client.getHorizonURL(), endpoint)

	return client.stream(ctx, url, func(data []byte) error {
		var ledger hProtocol.Ledger
		err = json.Unmarshal(data, &ledger)
		if err != nil {
			return errors.Wrap(err, "Error unmarshaling data")
		}
		handler(ledger)
		return nil
	})
}
