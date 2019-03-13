package horizonclient

import (
	"context"
	"net/http"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/errors"
)

func (c *Client) sendRequest(hr HorizonRequest, a interface{}) (err error) {
	endpoint, err := hr.BuildUrl()
	if err != nil {
		return
	}

	req, err := http.NewRequest("GET", c.HorizonURL+endpoint, nil)
	if err != nil {
		return errors.Wrap(err, "Error creating HTTP request")
	}
	req.Header.Set("X-Client-Name", "go-stellar-sdk")
	req.Header.Set("X-Client-Version", app.Version())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return
	}

	err = decodeResponse(resp, &a)
	return
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

	err = request.Stream(ctx, c.HorizonURL, handler)
	return
}

// Ledgers returns information about all ledgers.
// See https://www.stellar.org/developers/horizon/reference/endpoints/ledgers-all.html
func (c *Client) Ledgers(request LedgerRequest) (ledgers hProtocol.LedgersPage, err error) {
	err = c.sendRequest(request, &ledgers)
	return
}

// LedgerDetails returns information about a particular ledger for a given sequence number
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
