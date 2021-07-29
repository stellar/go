---
title: All Payments
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=payments&endpoint=all
---

This endpoint represents all payment-related [operations](../resources/operation.md) that are part
of validated [transactions](../resources/transaction.md). This endpoint can also be used in
[streaming](../streaming.md) mode so it is possible to use it to listen for new payments as they
get made in the DigitalBits network.

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
GET /payments{?cursor,limit,order,include_failed}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include payments of failed transactions in results. | `true` |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the payments in the response. | `transactions` |

### curl Example Request
# Retrieve the first 200 payments, ordered chronologically.

```sh
curl "https://frontier.testnet.digitalbits.io/payments?limit=200"
```
# Retrieve a page of payments to occur immediately before the transaction
# specified by the paging token "1234".

```sh
curl "https://frontier.testnet.digitalbits.io/payments?cursor=1234&order=desc"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.payments()
  .call()
  .then(function (paymentResults) {
    console.log(JSON.stringify(paymentResults.records))
  })
  .catch(function (err) {
    console.log(err)
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
  .cursor('now')
  .stream({
    onmessage: paymentHandler
  })
```

## Response

This endpoint responds with a list of payments. See [operation resource](../resources/operation.md) for more information about operations (and payment operations).

### Example Response

```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631874"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631874/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631874"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631874"
      }
    },
    "id": "1099511631874",
    "paging_token": "1099511631874",
    "transaction_successful": true,
    "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "create_account",
    "type_i": 0,
    "created_at": "2021-04-13T13:55:32Z",
    "transaction_hash": "081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9",
    "starting_balance": "101.0000000",
    "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "account": "GBPULLXKNDHPAP25N66JA4SH5SOQSNAIWKPVFTATMY6DDV43GBH2TUGV"
  },
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631875"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631875/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631875"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631875"
      }
    },
    "id": "1099511631875",
    "paging_token": "1099511631875",
    "transaction_successful": true,
    "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "create_account",
    "type_i": 0,
    "created_at": "2021-04-13T13:55:32Z",
    "transaction_hash": "081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9",
    "starting_balance": "101.0000000",
    "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "account": "GBQJVYYXDLTZ7RH6OWEQTTQ5G3A77WEZUFTFIYTXYOZUQHUI7NYOC6TO"
  }
]

```

### Example Streaming Event

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1103806599173"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/a8b22539d1f62825c527dbdfee8ba8d4faf701126021ccfa33bbe8cb149de9fd"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1103806599173/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1103806599173"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1103806599173"
    }
  },
  "id": "1103806599173",
  "paging_token": "1103806599173",
  "transaction_successful": true,
  "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "type": "create_account",
  "type_i": 0,
  "created_at": "2021-04-13T13:55:41Z",
  "transaction_hash": "a8b22539d1f62825c527dbdfee8ba8d4faf701126021ccfa33bbe8cb149de9fd",
  "starting_balance": "101.0000000",
  "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "account": "GAQS57T7RJHBZDSCX7E5KUI37CWUQ3CYIQ3KRMU6JEAQQYNCNXUL35NG"
}
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1103806599174"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/a8b22539d1f62825c527dbdfee8ba8d4faf701126021ccfa33bbe8cb149de9fd"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1103806599174/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1103806599174"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1103806599174"
    }
  },
  "id": "1103806599174",
  "paging_token": "1103806599174",
  "transaction_successful": true,
  "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "type": "create_account",
  "type_i": 0,
  "created_at": "2021-04-13T13:55:41Z",
  "transaction_hash": "a8b22539d1f62825c527dbdfee8ba8d4faf701126021ccfa33bbe8cb149de9fd",
  "starting_balance": "101.0000000",
  "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "account": "GCOMZ6JSBJU4WGIRTFETMZSCUUTUOPB7UJL3CQ6U4YFHCXD3PSGGF2X4"
}
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
