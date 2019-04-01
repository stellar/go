package horizonclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/support/errors"
)

// EffectHandler is a function that is called when a new effect is received
type EffectHandler func(effects.Base)

// BuildUrl creates the endpoint to be queried based on the data in the EffectRequest struct.
// If no data is set, it defaults to the build the URL for all effects
func (er EffectRequest) BuildUrl() (endpoint string, err error) {

	nParams := countParams(er.ForAccount, er.ForLedger, er.ForOperation, er.ForTransaction)

	if nParams > 1 {
		err = errors.New("Invalid request. Too many parameters")
	}

	if err != nil {
		return endpoint, err
	}

	endpoint = "effects"

	if er.ForAccount != "" {
		endpoint = fmt.Sprintf(
			"accounts/%s/effects",
			er.ForAccount,
		)
	}

	if er.ForLedger != "" {
		endpoint = fmt.Sprintf(
			"ledgers/%s/effects",
			er.ForLedger,
		)
	}

	if er.ForOperation != "" {
		endpoint = fmt.Sprintf(
			"operations/%s/effects",
			er.ForOperation,
		)
	}

	if er.ForTransaction != "" {
		endpoint = fmt.Sprintf(
			"transactions/%s/effects",
			er.ForTransaction,
		)
	}

	queryParams := addQueryParams(cursor(er.Cursor), limit(er.Limit), er.Order)
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

func (er EffectRequest) Stream(
	ctx context.Context,
	client *Client,
	handler func(interface{}),
) (err error) {

	url := fmt.Sprintf("%s/effects", client.getHorizonURL())
	return client.stream(ctx, url, func(data []byte) error {
		var effect effects.Base
		err = json.Unmarshal(data, &effect)
		if err != nil {
			return errors.Wrap(err, "Error unmarshaling data")
		}
		handler(effect)
		return nil
	})
}
