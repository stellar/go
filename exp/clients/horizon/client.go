package horizonclient

import (
	"github.com/stellar/go/support/errors"
)

func sendRequest(hr HorizonRequest, c Client, a interface{}) (err error) {
	endpoint, err := hr.BuildUrl()
	if err != nil {
		return
	}

	resp, err := c.HTTP.Get(c.HorizonURL + endpoint)
	if err != nil {

		return
	}

	err = decodeResponse(resp, &a)
	return
}

// AccountDetail returns information for a single account.
// See https://www.stellar.org/developers/horizon/reference/endpoints/accounts-single.html
func (c *Client) AccountDetail(request AccountRequest) (account Account, err error) {
	if request.AccountId == "" {
		err = errors.New("No account ID provided")
	}

	if err != nil {
		return
	}

	err = sendRequest(request, *c, &account)
	return
}

// AccountData returns a single data associated with a given account
// See https://www.stellar.org/developers/horizon/reference/endpoints/data-for-account.html
func (c *Client) AccountData(request AccountRequest) (accountData AccountData, err error) {
	if request.AccountId == "" || request.DataKey == "" {
		err = errors.New("Too few parameters")
	}

	if err != nil {
		return
	}

	err = sendRequest(request, *c, &accountData)
	return
}

// Effects returns effects(https://www.stellar.org/developers/horizon/reference/resources/effect.html)
// It can be used to return effects for an account, a ledger, an operation, a transaction and all effects on the network.
func (c *Client) Effects(request EffectRequest) (effects EffectsPage, err error) {
	err = sendRequest(request, *c, &effects)
	return
}

func (c *Client) Assets(request AssetRequest) (assets AssetsPage, err error) {
	err = sendRequest(request, *c, &assets)
	return
}
