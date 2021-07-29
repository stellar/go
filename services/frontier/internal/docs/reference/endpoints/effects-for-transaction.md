---
title: Effects for Transaction
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=effects&endpoint=for_transaction
---


This endpoint represents all [effects](../resources/effect.md) that occurred as a result of a given [transaction](../resources/transaction.md).

## Request

```
GET /transactions/{hash}/effects{?cursor,limit,order}
```

## Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `hash` | required, string | A transaction hash, hex-encoded, lowercase. | `8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit` | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/transactions/8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94/effects?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.effects()
  .forTransaction("8b77f4b2a5af0d6fab04dd91a4f0dcc5006034506aebdd86e543d27781372f94")
  .call()
  .then(function (effectResults) {
    //page 1
    console.log(JSON.stringify(effectResults.records))
  })
  .catch(function (err) {
    console.log(err)
  })

```

## Response

This endpoint responds with a list of effects on the ledger as a result of a given transaction. See [effect resource](../resources/effect.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1163936141313"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1163936141313-1"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1163936141313-1"
      }
    },
    "id": "0000001163936141313-0000000001",
    "paging_token": "1163936141313-1",
    "account": "GBUIN5WAS6QARKMUUXKWH2MLACV7XYOB3NQNBZN4ZLCWXCVD7KT5S43P",
    "type": "account_created",
    "type_i": 0,
    "created_at": "2021-04-13T13:57:06Z",
    "starting_balance": "101.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1163936141313"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1163936141313-2"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1163936141313-2"
      }
    },
    "id": "0000001163936141313-0000000002",
    "paging_token": "1163936141313-2",
    "account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "account_debited",
    "type_i": 3,
    "created_at": "2021-04-13T13:57:06Z",
    "asset_type": "native",
    "amount": "101.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1163936141313"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1163936141313-3"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1163936141313-3"
      }
    },
    "id": "0000001163936141313-0000000003",
    "paging_token": "1163936141313-3",
    "account": "GBUIN5WAS6QARKMUUXKWH2MLACV7XYOB3NQNBZN4ZLCWXCVD7KT5S43P",
    "type": "signer_created",
    "type_i": 10,
    "created_at": "2021-04-13T13:57:06Z",
    "weight": 1,
    "public_key": "GBUIN5WAS6QARKMUUXKWH2MLACV7XYOB3NQNBZN4ZLCWXCVD7KT5S43P",
    "key": ""
  }
]
```

## Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there are no effects for transaction whose hash matches the `hash` argument.
