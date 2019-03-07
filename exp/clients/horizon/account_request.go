package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the AccountRequest struct.
// If only AccountId is present, then the endpoint for account details is returned.
// If both AccounId and DataKey are present, then the endpoint for getting account data is returned
func (ar AccountRequest) BuildUrl() (endpoint string, err error) {

	nParams := countParams(ar.DataKey, ar.AccountId)

	if nParams >= 1 && ar.AccountId == "" {
		err = errors.New("Invalid request. Too few parameters")
	}

	if nParams <= 0 {
		err = errors.New("Invalid request. No parameters")
	}

	if err != nil {
		return endpoint, err
	}

	if ar.DataKey != "" && ar.AccountId != "" {
		endpoint = fmt.Sprintf(
			"accounts/%s/data/%s",
			ar.AccountId,
			ar.DataKey,
		)
	} else if ar.AccountId != "" {
		endpoint = fmt.Sprintf(
			"accounts/%s",
			ar.AccountId,
		)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
