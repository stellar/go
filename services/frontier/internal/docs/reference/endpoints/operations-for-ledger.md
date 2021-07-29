---
title: Operations for Ledger
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=operations&endpoint=for_ledger
---

This endpoint returns successful [operations](../resources/operation.md) that occurred in a given [ledger](../resources/ledger.md).

## Request

```
GET /ledgers/{sequence}/operations{?cursor,limit,order,include_failed}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `sequence` | required, number | Ledger Sequence | `957773` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit` | optional, number, default `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include operations of failed transactions in results. | `true` |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the operations in the response. | `transactions` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/ledgers/681637/operations?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.operations()
  .forLedger("957773")
  .call()
  .then(function (operationsResult) {
    console.log(JSON.stringify(operationsResult.records));
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a list of operations in a given ledger.  See [operation resource](../resources/operation.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4113603711995905"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/847b33d9e54a8884a9b9c1fd68dc5560e1c61e181155aafc1145e934cc12535d"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4113603711995905/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4113603711995905"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4113603711995905"
      }
    },
    "id": "4113603711995905",
    "paging_token": "4113603711995905",
    "transaction_successful": true,
    "source_account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "type": "change_trust",
    "type_i": 6,
    "created_at": "2021-06-16T09:08:23Z",
    "transaction_hash": "847b33d9e54a8884a9b9c1fd68dc5560e1c61e181155aafc1145e934cc12535d",
    "asset_type": "credit_alphanum4",
    "asset_code": "EUR",
    "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC",
    "limit": "1000.0000000",
    "trustee": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC",
    "trustor": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY"
  }
]

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no ledger whose ID matches the `id` argument.
