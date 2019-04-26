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

// BuildURL creates the endpoint to be queried based on the data in the EffectRequest struct.
// If no data is set, it defaults to the build the URL for all effects
func (er EffectRequest) BuildURL() (endpoint string, err error) {

	nParams := countParams(er.ForAccount, er.ForLedger, er.ForOperation, er.ForTransaction)

	if nParams > 1 {
		err = errors.New("invalid request: too many parameters")
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

// StreamEffects streams horizon effects. It can be used to stream all effects or account specific effects.
// Use context.WithCancel to stop streaming or context.Background() if you want to stream indefinitely.
// EffectHandler is a user-supplied function that is executed for each streamed effect received.
func (er EffectRequest) StreamEffects(
	ctx context.Context,
	client *Client,
	handler EffectHandler,
) (err error) {
	endpoint, err := er.BuildURL()
	if err != nil {
		return errors.Wrap(err, "unable to build endpoint")
	}

	url := fmt.Sprintf("%s%s", client.fixHorizonURL(), endpoint)
	return client.stream(ctx, url, func(data []byte) error {
		var effect effects.Base
		err = json.Unmarshal(data, &effect)
		if err != nil {
			return errors.Wrap(err, "error unmarshaling data")
		}
		handler(effect)
		return nil
	})
}
