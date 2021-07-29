---
title: Payments for Account
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=payments&endpoint=for_account
---

This endpoint responds with a collection of payment-related operations where the given
[account](../resources/account.md) was either the sender or receiver.

This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to
listen for new payments to or from an account as they get made in the DigitalBits network.
If called in streaming mode Frontier will start at the earliest known payment unless a `cursor` is
set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only
stream payments created since your request time.

The operations that can be returned in by this endpoint are:
- `create_account`
- `payment`
- `path_payment`
- `account_merge`

## Request

```
GET /accounts/{id}/payments{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `id` | required, string | The account id of the account used to constrain results. | `GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M` |
| `?cursor` | optional, default _null_ | A payment paging token specifying from where to begin results. When streaming this can be set to `now` to stream object created since your request time. | `8589934592` |
| `?limit` | optional, number, default `10` | Specifies the count of records at most to return. | `200` |
| `?order` | optional, string, default `asc` | Specifies order of returned results. `asc` means older payments first, `desc` mean newer payments first. | `desc` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include payments of failed transactions in results. | `true` |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the payments in the response. | `transactions` |

### curl Example Request
# Retrieve the 25 latest payments for a specific account.

```bash
curl "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/payments?limit=25&order=desc"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.payments()
  .forAccount("GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M")
  .call()
  .then(function (accountResult) {
    console.log(JSON.stingify(accountResult));
  })
  .catch(function (err) {
    console.error(err);
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var paymentHandler = function (paymentResponse) {
  console.log(paymentResponse);
};

var es = server.payments()
  .forAccount("GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M")
  .cursor('now')
  .stream({
    onmessage: paymentHandler
  })
```

## Response

This endpoint responds with a [page](../resources/page.md) of [payment operations](../resources/operation.md).

### Example Response

```json
{
  "records": [
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/operations/4109317334634497"
        },
        "transaction": {
          "href": "https://frontier.testnet.digitalbits.io/transactions/a12a1fc9f3a0d86e1bc2f04e086a027c84f290d30b5c5412b68addee1d039d03"
        },
        "effects": {
          "href": "https://frontier.testnet.digitalbits.io/operations/4109317334634497/effects"
        },
        "succeeds": {
          "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4109317334634497"
        },
        "precedes": {
          "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4109317334634497"
        }
      },
      "id": "4109317334634497",
      "paging_token": "4109317334634497",
      "transaction_successful": true,
      "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
      "type": "create_account",
      "type_i": 0,
      "created_at": "2021-06-16T07:33:52Z",
      "transaction_hash": "a12a1fc9f3a0d86e1bc2f04e086a027c84f290d30b5c5412b68addee1d039d03",
      "starting_balance": "10000.0000000",
      "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
      "account": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M"
    },
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/operations/4109381759143937"
        },
        "transaction": {
          "href": "https://frontier.testnet.digitalbits.io/transactions/f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec"
        },
        "effects": {
          "href": "https://frontier.testnet.digitalbits.io/operations/4109381759143937/effects"
        },
        "succeeds": {
          "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4109381759143937"
        },
        "precedes": {
          "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4109381759143937"
        }
      },
      "id": "4109381759143937",
      "paging_token": "4109381759143937",
      "transaction_successful": true,
      "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
      "type": "payment",
      "type_i": 1,
      "created_at": "2021-06-16T07:35:18Z",
      "transaction_hash": "f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec",
      "asset_type": "credit_alphanum4",
      "asset_code": "UAH",
      "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
      "from": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
      "to": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
      "amount": "10.0000000"
    }
  ]
}
```

### Example Streaming Event

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4109317334634497"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/a12a1fc9f3a0d86e1bc2f04e086a027c84f290d30b5c5412b68addee1d039d03"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4109317334634497/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4109317334634497"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4109317334634497"
    }
  },
  "id": "4109317334634497",
  "paging_token": "4109317334634497",
  "transaction_successful": true,
  "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "type": "create_account",
  "type_i": 0,
  "created_at": "2021-06-16T07:33:52Z",
  "transaction_hash": "a12a1fc9f3a0d86e1bc2f04e086a027c84f290d30b5c5412b68addee1d039d03",
  "starting_balance": "10000.0000000",
  "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "account": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M"
}
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4109381759143937"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4109381759143937/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4109381759143937"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4109381759143937"
    }
  },
  "id": "4109381759143937",
  "paging_token": "4109381759143937",
  "transaction_successful": true,
  "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "type": "payment",
  "type_i": 1,
  "created_at": "2021-06-16T07:35:18Z",
  "transaction_hash": "f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec",
  "asset_type": "credit_alphanum4",
  "asset_code": "UAH",
  "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "from": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "to": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
  "amount": "10.0000000"
}
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
