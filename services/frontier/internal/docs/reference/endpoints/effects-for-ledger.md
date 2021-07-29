---
title: Effects for Ledger
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=effects&endpoint=for_ledger
---

Effects are the specific ways that the ledger was changed by any operation.

This endpoint represents all [effects](../resources/effect.md) that occurred in the given [ledger](../resources/ledger.md).

## Request

```
GET /ledgers/{sequence}/effects{?cursor,limit,order}
```

## Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `sequence` | required, number | Ledger Sequence Number | `957773` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit` | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/ledgers/957773/effects?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.effects()
  .forLedger("957773")
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

This endpoint responds with a list of effects that occurred in the ledger. See [effect resource](../resources/effect.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4113603711995905"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4113603711995905-1"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4113603711995905-1"
      }
    },
    "id": "0004113603711995905-0000000001",
    "paging_token": "4113603711995905-1",
    "account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "type": "trustline_created",
    "type_i": 20,
    "created_at": "2021-06-16T09:08:23Z",
    "asset_type": "credit_alphanum4",
    "asset_code": "EUR",
    "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC",
    "limit": "1000.0000000"
  }
]
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there are no effects for a given ledger.
