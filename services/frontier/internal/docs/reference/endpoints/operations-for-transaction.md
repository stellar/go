---
title: Operations for Transaction
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=operations&endpoint=for_transaction
---

This endpoint represents successful [operations](../resources/operation.md) that are part of a given [transaction](../resources/transaction.md).

### Warning - failed transactions

The "Operations for Transaction" endpoint returns a list of operations in a successful or failed
transaction. Make sure to always check the operation status in this endpoint using
`transaction_successful` field!

## Request

```
GET /transactions/{hash}/operations{?cursor,limit,order}
```

## Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `hash` | required, string | A transaction hash, hex-encoded, lowercase. | `4a3365180521e16b478d9f0c9198b97a9434fc9cb07b34f83ecc32fc54d0ca8a` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit` | optional, number, default `10` | Maximum number of records to return. | `200` |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the operations in the response. | `transactions` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/transactions/4a3365180521e16b478d9f0c9198b97a9434fc9cb07b34f83ecc32fc54d0ca8a/operations?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.operations()
  .forTransaction("8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94")
  .call()
  .then(function (operationsResult) {
    console.log(JSON.stringify(operationsResult.records));
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a list of operations that are part of a given transaction. See [operation resource](../resources/operation.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1163936141314"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1163936141314/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1163936141314"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1163936141314"
      }
    },
    "id": "1163936141314",
    "paging_token": "1163936141314",
    "transaction_successful": true,
    "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "create_account",
    "type_i": 0,
    "created_at": "2021-04-13T13:57:06Z",
    "transaction_hash": "8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94",
    "starting_balance": "101.0000000",
    "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "account": "GCZEBR7M3M2EEBDFL77E4QTPABKYE4UF6UCD4CEOGLREM7YCHLJELB52"
  },
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1163936141315"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1163936141315/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1163936141315"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1163936141315"
      }
    },
    "id": "1163936141315",
    "paging_token": "1163936141315",
    "transaction_successful": true,
    "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "create_account",
    "type_i": 0,
    "created_at": "2021-04-13T13:57:06Z",
    "transaction_hash": "8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94",
    "starting_balance": "101.0000000",
    "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "account": "GBOAZSYJZJEUADWSBDWX2PIYIDUFMJINM3ZOTOSEIBGPAMYAW7HDGMMC"
  }
]

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no account whose ID matches the `hash` argument.
