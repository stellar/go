package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildUrl creates the endpoint to be queried based on the data in the TransactionRequest struct.
// If no data is set, it defaults to the build the URL for all transactions
func (tr TransactionRequest) BuildUrl() (endpoint string, err error) {
	nParams := countParams(tr.ForAccount, tr.ForLedger, tr.forTransactionHash)

	if nParams > 1 {
		return endpoint, errors.New("Invalid request. Too many parameters")
	}

	endpoint = "transactions"
	if tr.ForAccount != "" {
		endpoint = fmt.Sprintf("accounts/%s/transactions", tr.ForAccount)
	}
	if tr.ForLedger > 0 {
		endpoint = fmt.Sprintf("ledgers/%d/transactions", tr.ForLedger)
	}
	if tr.forTransactionHash != "" {
		endpoint = fmt.Sprintf("transactions/%s", tr.forTransactionHash)
	}

	queryParams := addQueryParams(cursor(tr.Cursor), limit(tr.Limit), tr.Order,
		includeFailed(tr.IncludeFailed))
	if queryParams != "" {
		endpoint = fmt.Sprintf("%s?%s", endpoint, queryParams)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
