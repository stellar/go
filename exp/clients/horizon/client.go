package horizonclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/manucorporat/sse"
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
	c.setClientAppHeaders(req)

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

// stream handles connections to endpoints that support streaming on an horizon server
func (c *Client) stream(
	ctx context.Context,
	streamURL string,
	handler func(data []byte) error,
) error {
	su, err := url.Parse(streamURL)
	if err != nil {
		return errors.Wrap(err, "Error parsing stream url")
	}

	query := su.Query()
	if query.Get("cursor") == "" {
		query.Set("cursor", "now")
	}

	for {
		// updates the url with new cursor
		su.RawQuery = query.Encode()
		req, err := http.NewRequest("GET", fmt.Sprintf("%s", su), nil)
		if err != nil {
			return errors.Wrap(err, "Error creating HTTP request")
		}
		req.Header.Set("Accept", "text/event-stream")
		// to do: confirm name and version
		c.setClientAppHeaders(req)

		// We can use c.HTTP here because we set Timeout per request not on the client. See sendRequest()
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return errors.Wrap(err, "Error sending HTTP request")
		}

		// Expected statusCode are 200-299
		if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
			return fmt.Errorf("Got bad HTTP status code %d", resp.StatusCode)
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		// Read events one by one. Break this loop when there is no more data to be
		// read from resp.Body (io.EOF).
	Events:
		for {
			// Read until empty line = event delimiter. The perfect solution would be to read
			// as many bytes as possible and forward them to sse.Decode. However this
			// requires much more complicated code.
			// We could also write our own `sse` package that works fine with streams directly
			// (github.com/manucorporat/sse is just using io/ioutils.ReadAll).
			var buffer bytes.Buffer
			nonEmptylinesRead := 0
			for {
				// Check if ctx is not cancelled
				select {
				case <-ctx.Done():
					return nil
				default:
					// Continue
				}

				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						// We catch EOF errors to handle two possible situations:
						// - The last line before closing the stream was not empty. This should never
						//   happen in Horizon as it always sends an empty line after each event.
						// - The stream was closed by the server/proxy because the connection was idle.
						//
						// In the former case, that (again) should never happen in Horizon, we need to
						// check if there are any events we need to decode. We do this in the `if`
						// statement below just in case if Horizon behaviour changes in a future.
						//
						// From spec:
						// > Once the end of the file is reached, the user agent must dispatch the
						// > event one final time, as defined below.
						if nonEmptylinesRead == 0 {
							break Events
						}
					} else {
						return errors.Wrap(err, "Error reading line")
					}
				}
				buffer.WriteString(line)

				if strings.TrimRight(line, "\n\r") == "" {
					break
				}

				nonEmptylinesRead++
			}

			events, err := sse.Decode(strings.NewReader(buffer.String()))
			if err != nil {
				return errors.Wrap(err, "Error decoding event")
			}

			// Right now len(events) should always be 1. This loop will be helpful after writing
			// new SSE decoder that can handle io.Reader without using ioutils.ReadAll().
			for _, event := range events {
				if event.Event != "message" {
					continue
				}

				// Update cursor with event ID
				if event.Id != "" {
					query.Set("cursor", event.Id)
				}

				switch data := event.Data.(type) {
				case string:
					err = handler([]byte(data))
					err = errors.Wrap(err, "Handler error")
				case []byte:
					err = handler(data)
					err = errors.Wrap(err, "Handler error")
				default:
					err = errors.New("Invalid event.Data type")
				}
				if err != nil {
					return err
				}
			}
		}
	}
}

func (c *Client) setClientAppHeaders(req *http.Request) {
	req.Header.Set("X-Client-Name", "go-stellar-sdk")
	req.Header.Set("X-Client-Version", app.Version())
	req.Header.Set("X-App-Name", c.AppName)
	req.Header.Set("X-App-Version", c.AppVersion)
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

	err = request.Stream(ctx, c, handler)
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

// Trades returns stellar trades (https://www.stellar.org/developers/horizon/reference/resources/trade.html)
// It can be used to return trades for an account, an offer and all trades on the network.
func (c *Client) Trades(request TradeRequest) (tds hProtocol.TradesPage, err error) {
	err = c.sendRequest(request, &tds)
	return
}

// StreamTrades streams executed trades. It can be used to stream all trades, trades for an account and
// trades for an offer. Use context.WithCancel to stop streaming or context.Background() if you want
// to stream indefinitely. TradeHandler is a user-supplied function that is executed for each streamed trade received.
func (c *Client) StreamTrades(ctx context.Context, request TradeRequest, handler TradeHandler) (err error) {
	err = request.StreamTrades(ctx, c, handler)
	return
}

// TradeAggregations returns stellar trade aggregations (https://www.stellar.org/developers/horizon/reference/resources/trade_aggregation.html)
func (c *Client) TradeAggregations(request TradeAggregationRequest) (tds hProtocol.TradeAggregationsPage, err error) {
	err = c.sendRequest(request, &tds)
	return
}

// StreamTransactions streams processed transactions. It can be used to stream all transactions and
// transactions for an account. Use context.WithCancel to stop streaming or context.Background()
// if you want to stream indefinitely. TransactionHandler is a user-supplied function that is executed for each streamed transaction received.
func (c *Client) StreamTransactions(ctx context.Context, request TransactionRequest, handler TransactionHandler) error {
	return request.StreamTransactions(ctx, c, handler)
}

// StreamEffects streams horizon effects. It can be used to stream all effects or account specific effects.
// Use context.WithCancel to stop streaming or context.Background() if you want to stream indefinitely.
// EffectHandler is a user-supplied function that is executed for each streamed transaction received.
func (c *Client) StreamEffects(ctx context.Context, request EffectRequest, handler EffectHandler) error {
	return request.StreamEffects(ctx, c, handler)
}

// StreamOffers streams offers processed by the Stellar network for an account. Use context.WithCancel
// to stop streaming or context.Background() if you want to stream indefinitely.
// OfferHandler is a user-supplied function that is executed for each streamed offer received.
func (c *Client) StreamOffers(ctx context.Context, request OfferRequest, handler OfferHandler) error {
	return request.StreamOffers(ctx, c, handler)
}

// StreamLedgers streams stellar ledgers. It can be used to stream all ledgers. Use context.WithCancel
// to stop streaming or context.Background() if you want to stream indefinitely.
// LedgerHandler is a user-supplied function that is executed for each streamed ledger received.
func (c *Client) StreamLedgers(ctx context.Context, request LedgerRequest, handler LedgerHandler) error {
	return request.StreamLedgers(ctx, c, handler)
}

// ensure that the horizon client implements ClientInterface
var _ ClientInterface = &Client{}
