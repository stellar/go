---
title: All Operations
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=operations&endpoint=all
---

This endpoint represents [operations](../resources/operation.md) that are part of successfully validated [transactions](../resources/transaction.md).
Please note that this endpoint returns operations that are part of failed transactions if `include_failed` parameter is `true`
and Frontier is ingesting failed transactions.
This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to listen as operations are processed in the DigitalBits network.
If called in streaming mode Frontier will start at the earliest known operation unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream operations created since your request time.

## Request

```
GET /operations{?cursor,limit,order,include_failed}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include operations of failed transactions in results. | `true` |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the operations in the response. | `transactions` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/operations?limit=200&order=desc"
```

### JavaScript Example Request

```js
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.operations()
  .call()
  .then(function (operationsResult) {
    //page 1
    console.log(JSON.stringify(operationsResult.records))
  })
  .catch(function (err) {
    console.log(err)
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var operationHandler = function (operationResponse) {
  console.log(operationResponse);
};

var es = server.operations()
  .cursor('now')
  .stream({
    onmessage: operationHandler
  })
```

## Response

This endpoint responds with a list of operations. See [operation resource](../resources/operation.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631881"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631881/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631881"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631881"
      }
    },
    "id": "1099511631881",
    "paging_token": "1099511631881",
    "transaction_successful": true,
    "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "create_account",
    "type_i": 0,
    "created_at": "2021-04-13T13:55:32Z",
    "transaction_hash": "081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9",
    "starting_balance": "101.0000000",
    "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "account": "GBASD74HP42AMZ5O3HFUUPI5VIZKPSDYTL5D7IYEWAF6JZUDSFOW5DCL"
  },
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631882"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631882/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631882"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631882"
      }
    },
    "id": "1099511631882",
    "paging_token": "1099511631882",
    "transaction_successful": true,
    "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "create_account",
    "type_i": 0,
    "created_at": "2021-04-13T13:55:32Z",
    "transaction_hash": "081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9",
    "starting_balance": "101.0000000",
    "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "account": "GBY5ORESVCMHI54ADKGGWXXXCSBOFHUWSQ3TUCMWOUVDRU7AENR6FRRT"
  }
]
```

### Example Streaming Event

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1176821043224"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/a5522b114b2722013de727b4b4d1eeae9f54493b5e174ce74b21a980884a2138"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1176821043224/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1176821043224"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1176821043224"
    }
  },
  "id": "1176821043224",
  "paging_token": "1176821043224",
  "transaction_successful": true,
  "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "type": "create_account",
  "type_i": 0,
  "created_at": "2021-04-13T13:57:23Z",
  "transaction_hash": "a5522b114b2722013de727b4b4d1eeae9f54493b5e174ce74b21a980884a2138",
  "starting_balance": "101.0000000",
  "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "account": "GAOV6XT7DSVYROAQ2QMM2PATMIPV7MSO26NEEOJSLXGDFCFNPWGRC2KU"
}
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1176821043225"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/a5522b114b2722013de727b4b4d1eeae9f54493b5e174ce74b21a980884a2138"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1176821043225/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1176821043225"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1176821043225"
    }
  },
  "id": "1176821043225",
  "paging_token": "1176821043225",
  "transaction_successful": true,
  "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "type": "create_account",
  "type_i": 0,
  "created_at": "2021-04-13T13:57:23Z",
  "transaction_hash": "a5522b114b2722013de727b4b4d1eeae9f54493b5e174ce74b21a980884a2138",
  "starting_balance": "101.0000000",
  "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "account": "GANOHGWMJSNJSH6IFEKMYSKOE6OYDDJACI47CQOPI7KMVSTUG55RDSKK"
}
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
