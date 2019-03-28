package horizonclient

import (
	"fmt"
	"testing"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTradeRequestBuildUrl(t *testing.T) {
	tr := TradeRequest{}
	endpoint, err := tr.BuildUrl()

	// It should return valid all trades endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "trades", endpoint)

	tr = TradeRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err = tr.BuildUrl()

	// It should return valid account trades endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/trades", endpoint)

	tr = TradeRequest{ForOfferID: "123"}
	endpoint, err = tr.BuildUrl()

	// It should return valid offer trades endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "offers/123/trades", endpoint)

	tr = TradeRequest{Cursor: "123"}
	endpoint, err = tr.BuildUrl()

	// It should return valid trades endpoint and no errors
	require.NoError(t, err)
	assert.Equal(t, "trades?cursor=123", endpoint)

	tr = TradeRequest{ForOfferID: "123", ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	endpoint, err = tr.BuildUrl()

	// error case: too many parameters for building any operation endpoint
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Invalid request. Too many parameters")
	}

	tr = TradeRequest{Cursor: "123456", Limit: 30, Order: OrderAsc}
	endpoint, err = tr.BuildUrl()
	// It should return valid all trades endpoint with query params and no errors
	require.NoError(t, err)
	assert.Equal(t, "trades?cursor=123456&limit=30&order=asc", endpoint)

}

func ExampleClient_Trades() {

	client := DefaultPublicNetClient
	// Find all trades
	tr := TradeRequest{Cursor: "123456", Limit: 30, Order: OrderAsc}
	trades, err := client.Trades(tr)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(trades)
}

func TestTradesRequest(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	tradeRequest := TradeRequest{}

	// all trades
	hmock.On(
		"GET",
		"https://localhost/trades",
	).ReturnString(200, tradesResponse)

	trades, err := client.Trades(tradeRequest)
	if assert.NoError(t, err) {
		assert.IsType(t, trades, hProtocol.TradesPage{})
		links := trades.Links
		assert.Equal(t, links.Self.Href, "https://horizon-testnet.stellar.org/trades?cursor=&limit=2&order=desc")

		assert.Equal(t, links.Next.Href, "https://horizon-testnet.stellar.org/trades?cursor=2099298409914407-0&limit=2&order=desc")

		assert.Equal(t, links.Prev.Href, "https://horizon-testnet.stellar.org/trades?cursor=2099319884746791-0&limit=2&order=asc")

		trade := trades.Embedded.Records[0]
		assert.IsType(t, trade, hProtocol.Trade{})
		assert.Equal(t, trade.ID, "2099319884746791-0")
		assert.Equal(t, trade.BaseAmount, "2.4104452")
		assert.Equal(t, trade.CounterAmount, "0.0973412")
		assert.Equal(t, trade.OfferID, "3698823")
		assert.Equal(t, trade.BaseIsSeller, false)
	}

	tradeRequest = TradeRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/trades",
	).ReturnString(200, tradesResponse)

	trades, err = client.Trades(tradeRequest)
	if assert.NoError(t, err) {
		assert.IsType(t, trades, hProtocol.TradesPage{})
	}

	// too many parameters
	tradeRequest = TradeRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU", ForOfferID: "123"}
	hmock.On(
		"GET",
		"https://localhost/trades",
	).ReturnString(200, "")

	_, err = client.Trades(tradeRequest)
	// error case
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Too many parameters")
	}
}

var tradesResponse = `{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/trades?cursor=&limit=2&order=desc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/trades?cursor=2099298409914407-0&limit=2&order=desc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/trades?cursor=2099319884746791-0&limit=2&order=asc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": ""
          },
          "base": {
            "href": "https://horizon-testnet.stellar.org/accounts/GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C"
          },
          "counter": {
            "href": "https://horizon-testnet.stellar.org/accounts/GCYN7MI6VXVRP74KR6MKBAW2ELLCXL6QCY5H4YQ62HVWZWMCE6Y232UC"
          },
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/2099319884746791"
          }
        },
        "id": "2099319884746791-0",
        "paging_token": "2099319884746791-0",
        "ledger_close_time": "2019-03-28T10:45:28Z",
        "offer_id": "3698823",
        "base_offer_id": "4613785338312134695",
        "base_account": "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C",
        "base_amount": "2.4104452",
        "base_asset_type": "credit_alphanum4",
        "base_asset_code": "DSQ",
        "base_asset_issuer": "GBDQPTQJDATT7Z7EO4COS4IMYXH44RDLLI6N6WIL5BZABGMUOVMLWMQF",
        "counter_offer_id": "3698823",
        "counter_account": "GCYN7MI6VXVRP74KR6MKBAW2ELLCXL6QCY5H4YQ62HVWZWMCE6Y232UC",
        "counter_amount": "0.0973412",
        "counter_asset_type": "credit_alphanum4",
        "counter_asset_code": "USD",
        "counter_asset_issuer": "GAA4MFNZGUPJAVLWWG6G5XZJFZDHLKQNG3Q6KB24BAD6JHNNVXDCF4XG",
        "base_is_seller": false,
        "price": {
          "n": 2000000,
          "d": 49525693
        }
      },
      {
        "_links": {
          "self": {
            "href": ""
          },
          "base": {
            "href": "https://horizon-testnet.stellar.org/accounts/GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C"
          },
          "counter": {
            "href": "https://horizon-testnet.stellar.org/accounts/GCYN7MI6VXVRP74KR6MKBAW2ELLCXL6QCY5H4YQ62HVWZWMCE6Y232UC"
          },
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/2099298409914407"
          }
        },
        "id": "2099298409914407-0",
        "paging_token": "2099298409914407-0",
        "ledger_close_time": "2019-03-28T10:45:02Z",
        "offer_id": "3698823",
        "base_offer_id": "4613785316837302311",
        "base_account": "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C",
        "base_amount": "89.3535843",
        "base_asset_type": "credit_alphanum4",
        "base_asset_code": "DSQ",
        "base_asset_issuer": "GBDQPTQJDATT7Z7EO4COS4IMYXH44RDLLI6N6WIL5BZABGMUOVMLWMQF",
        "counter_offer_id": "3698823",
        "counter_account": "GCYN7MI6VXVRP74KR6MKBAW2ELLCXL6QCY5H4YQ62HVWZWMCE6Y232UC",
        "counter_amount": "3.6083729",
        "counter_asset_type": "credit_alphanum4",
        "counter_asset_code": "USD",
        "counter_asset_issuer": "GAA4MFNZGUPJAVLWWG6G5XZJFZDHLKQNG3Q6KB24BAD6JHNNVXDCF4XG",
        "base_is_seller": false,
        "price": {
          "n": 2000000,
          "d": 49525693
        }
      }
    ]
  }
}`
