package horizonclient

import "github.com/stellar/go/support/errors"

// BuildUrl returns the url for getting metrics about a running horizon instance
func (mr metricsRequest) BuildUrl() (endpoint string, err error) {
	endpoint = mr.endpoint
	if endpoint == "" {
		err = errors.New("Invalid request. Too few parameters")
	}

	return
}
