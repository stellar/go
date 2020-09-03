package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildURL creates the endpoint to be queried based on the data in the TransactionRequest struct.
// If no data is set, it defaults to the build the URL for all transactions
func (cbr ClaimableBalanceRequest) BuildURL() (endpoint string, err error) {
	endpoint = "claimable_balances"

	// According to CAP-23, only one filter parameter is allowed.
	nParams := countParams(cbr.Asset, cbr.Claimant, cbr.Sponsor)
	if nParams > 1 {
		return endpoint, errors.New("invalid request: too many parameters")
	}

	// We can also specify an ID in addition to the filters, though.
	if cbr.ID != "" {
		endpoint = fmt.Sprintf("%s/%s", endpoint, cbr.ID)
	}

	// queryParams := ""
	// if queryParams != "" {
	//  endpoint = fmt.Sprintf("%s?%s", endpoint, queryParams)
	// }

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
