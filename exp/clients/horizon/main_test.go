package horizonclient

import (
	"fmt"
	"testing"

	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
)

func ExampleClient_AccountDetail() {

	client := DefaultPublicNetClient
	accountRequest := AccountRequest{AccountId: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}

	account, err := client.AccountDetail(accountRequest)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Print(account)
}

func ExampleClient_Effects() {

	client := DefaultPublicNetClient
	// effects for an account
	effectRequest := EffectRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	effect, err := client.Effects(effectRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(effect)

	// all effects
	effectRequest = EffectRequest{}
	effect, err = client.Effects(effectRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(effect)
}

func ExampleClient_Assets() {

	client := DefaultPublicNetClient
	// assets for asset issuer
	assetRequest := AssetRequest{ForAssetIssuer: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	asset, err := client.Assets(assetRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(asset)

	// all assets
	assetRequest = AssetRequest{}
	asset, err = client.Assets(assetRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(asset)
}

func TestAccountDetail(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	// no parameters
	accountRequest := AccountRequest{}
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
	).ReturnString(200, accountResponse)

	_, err := client.AccountDetail(accountRequest)
	// error case: no account id
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "No account ID provided")
	}

	// wrong parameters
	accountRequest = AccountRequest{DataKey: "test"}
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
	).ReturnString(200, accountResponse)

	_, err = client.AccountDetail(accountRequest)
	// error case: no account id
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "No account ID provided")
	}

	accountRequest = AccountRequest{AccountId: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}

	// happy path
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
	).ReturnString(200, accountResponse)

	account, err := client.AccountDetail(accountRequest)

	if assert.NoError(t, err) {
		assert.Equal(t, account.ID, "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU")
		assert.Equal(t, account.PT, "1")
		assert.Equal(t, account.Signers[0].Key, "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU")
		assert.Equal(t, account.Signers[0].Type, "ed25519_public_key")
		assert.Equal(t, account.Data["test"], "dGVzdA==")
		balance, err := account.GetNativeBalance()
		assert.Nil(t, err)
		assert.Equal(t, balance, "9999.9999900")
	}

	// failure response
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
	).ReturnString(404, notFoundResponse)

	account, err = client.AccountDetail(accountRequest)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Horizon error")
		horizonError, ok := err.(*Error)
		assert.Equal(t, ok, true)
		assert.Equal(t, horizonError.Problem.Title, "Resource Missing")
	}

	// connection error
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
	).ReturnError("http.Client error")

	_, err = client.AccountDetail(accountRequest)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "http.Client error")
		_, ok := err.(*Error)
		assert.Equal(t, ok, false)
	}
}

func TestAccountData(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	// no parameters
	accountRequest := AccountRequest{}
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/data/test",
	).ReturnString(200, accountResponse)

	_, err := client.AccountData(accountRequest)
	// error case: few parameters
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Too few parameters")
	}

	// wrong parameters
	accountRequest = AccountRequest{AccountId: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/data/test",
	).ReturnString(200, accountResponse)

	_, err = client.AccountData(accountRequest)
	// error case: few parameters
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Too few parameters")
	}

	accountRequest = AccountRequest{AccountId: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU", DataKey: "test"}

	// happy path
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/data/test",
	).ReturnString(200, accountData)

	data, err := client.AccountData(accountRequest)
	if assert.NoError(t, err) {
		assert.Equal(t, data.Value, "dGVzdA==")
	}

}

func TestEffectsRequest(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	effectRequest := EffectRequest{}

	// all effects
	hmock.On(
		"GET",
		"https://localhost/effects",
	).ReturnString(200, effectsResponse)

	effects, err := client.Effects(effectRequest)
	if assert.NoError(t, err) {
		assert.IsType(t, effects, EffectsPage{})

	}

	effectRequest = EffectRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"}
	hmock.On(
		"GET",
		"https://localhost/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/effects",
	).ReturnString(200, effectsResponse)

	effects, err = client.Effects(effectRequest)
	if assert.NoError(t, err) {
		assert.IsType(t, effects, EffectsPage{})
	}

	// too many parameters
	effectRequest = EffectRequest{ForAccount: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU", ForLedger: "123"}
	hmock.On(
		"GET",
		"https://localhost/effects",
	).ReturnString(200, effectsResponse)

	_, err = client.Effects(effectRequest)
	// error case
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Too many parameters")
	}

}

func TestAssetsRequest(t *testing.T) {
	hmock := httptest.NewClient()
	client := &Client{
		HorizonURL: "https://localhost/",
		HTTP:       hmock,
	}

	assetRequest := AssetRequest{}

	// all assets
	hmock.On(
		"GET",
		"https://localhost/assets",
	).ReturnString(200, assetsResponse)

	assets, err := client.Assets(assetRequest)
	if assert.NoError(t, err) {
		assert.IsType(t, assets, AssetsPage{})
		record := assets.Embedded.Records[0]
		assert.Equal(t, record.Asset.Code, "ABC")
		assert.Equal(t, record.Asset.Issuer, "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU")
		assert.Equal(t, record.PT, "1")
		assert.Equal(t, record.NumAccounts, int32(3))
		assert.Equal(t, record.Amount, "105.0000000")
		assert.Equal(t, record.Flags.AuthRevocable, false)
		assert.Equal(t, record.Flags.AuthRequired, true)
		assert.Equal(t, record.Flags.AuthImmutable, false)
	}

}

var accountResponse = `{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"
    },
    "transactions": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/transactions{?cursor,limit,order}",
      "templated": true
    },
    "operations": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/operations{?cursor,limit,order}",
      "templated": true
    },
    "payments": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/payments{?cursor,limit,order}",
      "templated": true
    },
    "effects": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/effects{?cursor,limit,order}",
      "templated": true
    },
    "offers": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/offers{?cursor,limit,order}",
      "templated": true
    },
    "trades": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/trades{?cursor,limit,order}",
      "templated": true
    },
    "data": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU/data/{key}",
      "templated": true
    }
  },
  "id": "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
  "paging_token": "1",
  "account_id": "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
  "sequence": "9865509814140929",
  "subentry_count": 1,
  "thresholds": {
    "low_threshold": 0,
    "med_threshold": 0,
    "high_threshold": 0
  },
  "flags": {
    "auth_required": false,
    "auth_revocable": false,
    "auth_immutable": false
  },
  "balances": [
    {
      "balance": "9999.9999900",
      "buying_liabilities": "0.0000000",
      "selling_liabilities": "0.0000000",
      "asset_type": "native"
    }
  ],
  "signers": [
    {
      "public_key": "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
      "weight": 1,
      "key": "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
      "type": "ed25519_public_key"
    }
  ],
  "data": {
    "test": "dGVzdA=="
  }
}`

var notFoundResponse = `{
  "type": "https://stellar.org/horizon-errors/not_found",
  "title": "Resource Missing",
  "status": 404,
  "detail": "The resource at the url requested was not found.  This is usually occurs for one of two reasons:  The url requested is not valid, or no data in our database could be found with the parameters provided.",
  "instance": "horizon-live-001/61KdRW8tKi-18408110"
}`

var accountData = `{
  "value": "dGVzdA=="
}`

var effectsResponse = `{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/operations/43989725060534273/effects?cursor=&limit=10&order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/operations/43989725060534273/effects?cursor=43989725060534273-3&limit=10&order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/operations/43989725060534273/effects?cursor=43989725060534273-1&limit=10&order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/43989725060534273"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc&cursor=43989725060534273-1"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc&cursor=43989725060534273-1"
          }
        },
        "id": "0043989725060534273-0000000001",
        "paging_token": "43989725060534273-1",
        "account": "GANHAS5OMPLKD6VYU4LK7MBHSHB2Q37ZHAYWOBJRUXGDHMPJF3XNT45Y",
        "type": "account_debited",
        "type_i": 3,
        "created_at": "2018-07-27T21:00:12Z",
        "asset_type": "native",
        "amount": "9999.9999900"
      },
      {
        "_links": {
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/43989725060534273"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc&cursor=43989725060534273-2"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc&cursor=43989725060534273-2"
          }
        },
        "id": "0043989725060534273-0000000002",
        "paging_token": "43989725060534273-2",
        "account": "GBO7LQUWCC7M237TU2PAXVPOLLYNHYCYYFCLVMX3RBJCML4WA742X3UB",
        "type": "account_credited",
        "type_i": 2,
        "created_at": "2018-07-27T21:00:12Z",
        "asset_type": "native",
        "amount": "9999.9999900"
      },
      {
        "_links": {
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/43989725060534273"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc&cursor=43989725060534273-3"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc&cursor=43989725060534273-3"
          }
        },
        "id": "0043989725060534273-0000000003",
        "paging_token": "43989725060534273-3",
        "account": "GANHAS5OMPLKD6VYU4LK7MBHSHB2Q37ZHAYWOBJRUXGDHMPJF3XNT45Y",
        "type": "account_removed",
        "type_i": 1,
        "created_at": "2018-07-27T21:00:12Z"
      }
    ]
  }
}`

var assetsResponse = `{
    "_links": {
        "self": {
            "href": "https://horizon-testnet.stellar.org/assets?cursor=&limit=1&order=desc"
        },
        "next": {
            "href": "https://horizon-testnet.stellar.org/assets?cursor=ABC_GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU_credit_alphanum12&limit=1&order=desc"
        },
        "prev": {
            "href": "https://horizon-testnet.stellar.org/assets?cursor=ABC_GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU_credit_alphanum12&limit=1&order=asc"
        }
    },
    "_embedded": {
        "records": [
            {
                "_links": {
                    "toml": {
                        "href": ""
                    }
                },
                "asset_type": "credit_alphanum12",
                "asset_code": "ABC",
                "asset_issuer": "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
                "paging_token": "1",
                "amount": "105.0000000",
                "num_accounts": 3,
                "flags": {
                    "auth_required": true,
                    "auth_revocable": false,
                    "auth_immutable": false
                }
            }
        ]
    }
}`
