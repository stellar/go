package horizonclient

import "github.com/stellar/go/support/errors"

// BuildURL returns the url for getting fee stats about a running horizon instance
func (fr feeStatsRequest) BuildURL() (endpoint string, err error) {
	endpoint = fr.endpoint
	if endpoint == "" {
		err = errors.New("Invalid request. Too few parameters")
	}

	return
}
