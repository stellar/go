package horizonclient

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/errors"
)

func (c *Client) sendRequest(hr HorizonRequest, a interface{}) (err error) {
	endpoint, err := hr.BuildUrl()
	if err != nil {
		return
	}

	c.HorizonURL = c.getHorizonURL()
	var req *http.Request
	// check if it is a submitRequest
	_, ok := hr.(submitRequest)
	if ok {
		req, err = http.NewRequest("POST", c.HorizonURL+endpoint, nil)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	} else {
		req, err = http.NewRequest("GET", c.HorizonURL+endpoint, nil)
	}

	if err != nil {
		return errors.Wrap(err, "Error creating HTTP request")
	}
	req.Header.Set("X-Client-Name", "go-stellar-sdk")
	req.Header.Set("X-Client-Version", app.Version())

	if c.horizonTimeOut == 0 {
		c.horizonTimeOut = HorizonTimeOut
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*c.horizonTimeOut)

	resp, err := c.HTTP.Do(req.WithContext(ctx))
	if err != nil {
		cancel()
		return
	}

	err = decodeResponse(resp, &a)
	cancel()
	return
}

// getHorizonUrl strips all slashes(/) at the end of HorizonURL if any, then adds a single slash
func (c *Client) getHorizonURL() string {
	return strings.TrimRight(c.HorizonURL, "/") + "/"
}

// SetHorizonTimeOut allows users to set the number of seconds before a horizon request is cancelled.
func (c *Client) SetHorizonTimeOut(t uint) *Client {
	c.horizonTimeOut = time.Duration(t)
	return c
}

// GetHorizonTimeOut returns the current timeout for an horizon client
func (c *Client) GetHorizonTimeOut() time.Duration {
	return c.horizonTimeOut
}

// AccountDetail returns information for a single account.
// See https://www.stellar.org/developers/horizon/reference/endpoints/accounts-single.html
func (c *Client) AccountDetail(request AccountRequest) (account hProtocol.Account, err error) {
	if request.AccountId == "" {
		err = errors.New("No account ID provided")
	}

	if err != nil {
		return
	}

	err = c.sendRequest(request, &account)
	return
}

// AccountData returns a single data associated with a given account
// See https://www.stellar.org/developers/horizon/reference/endpoints/data-for-account.html
func (c *Client) AccountData(request AccountRequest) (accountData hProtocol.AccountData, err error) {
	if request.AccountId == "" || request.DataKey == "" {
		err = errors.New("Too few parameters")
	}

	if err != nil {
		return
	}

	err = c.sendRequest(request, &accountData)
	return
}

// Effects returns effects(https://www.stellar.org/developers/horizon/reference/resources/effect.html)
// It can be used to return effects for an account, a ledger, an operation, a transaction and all effects on the network.
func (c *Client) Effects(request EffectRequest) (effects hProtocol.EffectsPage, err error) {
	err = c.sendRequest(request, &effects)
	return
}

// Assets returns asset information.
// See https://www.stellar.org/developers/horizon/reference/endpoints/assets-all.html
func (c *Client) Assets(request AssetRequest) (assets hProtocol.AssetsPage, err error) {
	err = c.sendRequest(request, &assets)
	return
}

// Stream is for endpoints that support streaming
func (c *Client) Stream(ctx context.Context, request StreamRequest, handler func(interface{})) (err error) {

	err = request.Stream(ctx, c.getHorizonURL(), handler)
	return
}

// Ledgers returns information about all ledgers.
// See https://www.stellar.org/developers/horizon/reference/endpoints/ledgers-all.html
func (c *Client) Ledgers(request LedgerRequest) (ledgers hProtocol.LedgersPage, err error) {
	err = c.sendRequest(request, &ledgers)
	return
}

// LedgerDetail returns information about a particular ledger for a given sequence number
// See https://www.stellar.org/developers/horizon/reference/endpoints/ledgers-single.html
func (c *Client) LedgerDetail(sequence uint32) (ledger hProtocol.Ledger, err error) {
	if sequence <= 0 {
		err = errors.New("Invalid sequence number provided")
	}

	if err != nil {
		return
	}

	request := LedgerRequest{forSequence: sequence}

	err = c.sendRequest(request, &ledger)
	return
}

// Metrics returns monitoring information about a horizon server
// See https://www.stellar.org/developers/horizon/reference/endpoints/metrics.html
func (c *Client) Metrics() (metrics hProtocol.Metrics, err error) {
	request := metricsRequest{endpoint: "metrics"}
	err = c.sendRequest(request, &metrics)
	return
}

// FeeStats returns information about fees in the last 5 ledgers.
// See https://www.stellar.org/developers/horizon/reference/endpoints/fee-stats.html
func (c *Client) FeeStats() (feestats hProtocol.FeeStats, err error) {
	request := feeStatsRequest{endpoint: "fee_stats"}
	err = c.sendRequest(request, &feestats)
	return
}

// Offers returns information about offers made on the SDEX.
// See https://www.stellar.org/developers/horizon/reference/endpoints/offers-for-account.html
func (c *Client) Offers(request OfferRequest) (offers hProtocol.OffersPage, err error) {
	if request.ForAccount == "" {
		err = errors.New("`ForAccount` parameter required")
	}

	if err != nil {
		return
	}

	err = c.sendRequest(request, &offers)
	return
}

// Operations returns stellar operations (https://www.stellar.org/developers/horizon/reference/resources/operation.html)
// It can be used to return operations for an account, a ledger, a transaction and all operations on the network.
func (c *Client) Operations(request OperationRequest) (ops operations.OperationsPage, err error) {
	err = c.sendRequest(request.setEndpoint("operations"), &ops)
	return
}

// OperationDetail returns a single stellar operations (https://www.stellar.org/developers/horizon/reference/resources/operation.html)
// for a given operation id
func (c *Client) OperationDetail(id string) (ops operations.Operation, err error) {
	if id == "" {
		return ops, errors.New("Invalid operation id provided")
	}

	request := OperationRequest{forOperationId: id, endpoint: "operations"}

	var record interface{}

	err = c.sendRequest(request, &record)
	if err != nil {
		return ops, errors.Wrap(err, "Sending request to horizon")
	}

	var baseRecord operations.Base
	dataString, err := json.Marshal(record)
	if err != nil {
		return ops, errors.Wrap(err, "Marshaling json")
	}
	if err = json.Unmarshal(dataString, &baseRecord); err != nil {
		return ops, errors.Wrap(err, "Unmarshaling json")
	}

	ops, err = operations.UnmarshalOperation(baseRecord.GetType(), dataString)
	return ops, errors.Wrap(err, "Unmarshaling to the correct operation type")
}

// SubmitTransaction submits a transaction to the network. err can be either error object or horizon.Error object.
// See https://www.stellar.org/developers/horizon/reference/endpoints/transactions-create.html
func (c *Client) SubmitTransaction(transactionXdr string) (txSuccess hProtocol.TransactionSuccess,
	err error) {
	request := submitRequest{endpoint: "transactions", transactionXdr: transactionXdr}
	err = c.sendRequest(request, &txSuccess)
	return

}

// Transactions returns stellar transactions (https://www.stellar.org/developers/horizon/reference/resources/transaction.html)
// It can be used to return transactions for an account, a ledger,and all transactions on the network.
func (c *Client) Transactions(request TransactionRequest) (txs hProtocol.TransactionsPage, err error) {
	err = c.sendRequest(request, &txs)
	return
}

// TransactionDetail returns information about a particular transaction for a given transaction hash
// See https://www.stellar.org/developers/horizon/reference/endpoints/transactions-single.html
func (c *Client) TransactionDetail(txHash string) (tx hProtocol.Transaction, err error) {
	if txHash == "" {
		return tx, errors.New("No transaction hash provided")
	}

	request := TransactionRequest{forTransactionHash: txHash}
	err = c.sendRequest(request, &tx)
	return
}

// OrderBook returns the orderbook for an asset pair (https://www.stellar.org/developers/horizon/reference/resources/orderbook.html)
func (c *Client) OrderBook(request OrderBookRequest) (obs hProtocol.OrderBookSummary, err error) {
	err = c.sendRequest(request, &obs)
	return
}

// Paths returns the available paths to make a payment. See https://www.stellar.org/developers/horizon/reference/endpoints/path-finding.html
func (c *Client) Paths(request PathsRequest) (paths hProtocol.PathsPage, err error) {
	err = c.sendRequest(request, &paths)
	return
}

// Payments returns stellar account_merge, create_account, path payment and payment operations.
// It can be used to return payments for an account, a ledger, a transaction and all payments on the network.
func (c *Client) Payments(request OperationRequest) (ops operations.OperationsPage, err error) {
	err = c.sendRequest(request.setEndpoint("payments"), &ops)
	return
}

// ensure that the horizon client implements ClientInterface
var _ ClientInterface = &Client{}
