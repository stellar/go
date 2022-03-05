package horizonclient

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stellar/go/support/errors"
)

// Creates the URL to either request a specific claimable balance (CB) by ID, or
// request all CBs, possibly filtered by asset, claimant, or sponsor.
func (cbr ClaimableBalanceRequest) BuildURL() (endpoint string, err error) {
	endpoint = "claimable_balances"

	// Only one filter parameter is allowed, and you can't mix an ID query and
	// filters.
	nParams := countParams(cbr.Asset, cbr.Claimant, cbr.Sponsor, cbr.ID)
	if cbr.ID != "" && nParams > 1 {
		return endpoint, errors.New("invalid request: too many parameters")
	}

	if cbr.ID != "" {
		endpoint = fmt.Sprintf("%s/%s", endpoint, cbr.ID)
	} else {
		params := map[string]string{
			"claimant": cbr.Claimant,
			"sponsor":  cbr.Sponsor,
			"asset":    cbr.Asset,
			"cursor":   cbr.Cursor,
		}
		if (cbr.Limit) > 0 {
			params["limit"] = strconv.FormatUint(uint64(cbr.Limit), 10)
		}
		queryParams := addQueryParams(
			params,
		)

		endpoint = fmt.Sprintf("%s?%s", endpoint, queryParams)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}

// HTTPRequest returns the http request for the claimable balances endpoint
func (cbr ClaimableBalanceRequest) HTTPRequest(horizonURL string) (*http.Request, error) {
	endpoint, err := cbr.BuildURL()
	if err != nil {
		return nil, err
	}

	return http.NewRequest("GET", horizonURL+endpoint, nil)
}
