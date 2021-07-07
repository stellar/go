package frontierclient

import (
	"testing"

	"github.com/xdbfoundation/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
)

func TestRoot(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		FrontierURL: "https://localhost/",
		HTTP:       hmock,
	}

	// happy path
	hmock.On(
		"GET",
		"https://localhost/",
	).ReturnString(200, rootResponse)

	root, err := client.Root()
	if assert.NoError(t, err) {
		assert.Equal(t, root.FrontierVersion, "0.17.6-unstable-bc999a67d0b2413d8abd76153a56733c7d517484")
		assert.Equal(t, root.DigitalBitsCoreVersion, "digitalbits-core 11.0.0 (236f831521b6724c0ae63906416faa997ef27e19)")
		assert.Equal(t, root.FrontierSequence, int32(84959))
		assert.Equal(t, root.NetworkPassphrase, "Test SDF Network ; September 2015")
	}

	// failure response
	hmock.On(
		"GET",
		"https://localhost/",
	).ReturnString(404, notFoundResponse)

	_, err = client.Root()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "frontier error")
		frontierError, ok := err.(*Error)
		assert.Equal(t, ok, true)
		assert.Equal(t, frontierError.Problem.Title, "Resource Missing")
	}

	// connection error
	hmock.On(
		"GET",
		"https://localhost/",
	).ReturnError("http.Client error")

	_, err = client.Root()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "http.Client error")
		_, ok := err.(*Error)
		assert.Equal(t, ok, false)
	}
}

var rootResponse = `{
  "_links": {
    "account": {
      "href": "https://frontier-testnet.digitalbits.org/accounts/{account_id}",
      "templated": true
    },
    "account_transactions": {
      "href": "https://frontier-testnet.digitalbits.org/accounts/{account_id}/transactions{?cursor,limit,order}",
      "templated": true
    },
    "assets": {
      "href": "https://frontier-testnet.digitalbits.org/assets{?asset_code,asset_issuer,cursor,limit,order}",
      "templated": true
    },
    "friendbot": {
      "href": "https://friendbot.digitalbits.org/{?addr}",
      "templated": true
    },
    "metrics": {
      "href": "https://frontier-testnet.digitalbits.org/metrics"
    },
    "order_book": {
      "href": "https://frontier-testnet.digitalbits.org/order_book{?selling_asset_type,selling_asset_code,selling_asset_issuer,buying_asset_type,buying_asset_code,buying_asset_issuer,limit}",
      "templated": true
    },
    "self": {
      "href": "https://frontier-testnet.digitalbits.org/"
    },
    "transaction": {
      "href": "https://frontier-testnet.digitalbits.org/transactions/{hash}",
      "templated": true
    },
    "transactions": {
      "href": "https://frontier-testnet.digitalbits.org/transactions{?cursor,limit,order}",
      "templated": true
    }
  },
  "frontier_version": "0.17.6-unstable-bc999a67d0b2413d8abd76153a56733c7d517484",
  "core_version": "digitalbits-core 11.0.0 (236f831521b6724c0ae63906416faa997ef27e19)",
  "history_latest_ledger": 84959,
  "history_elder_ledger": 1,
  "core_latest_ledger": 84959,
  "network_passphrase": "Test SDF Network ; September 2015",
  "current_protocol_version": 10,
  "core_supported_protocol_version": 11
}`
