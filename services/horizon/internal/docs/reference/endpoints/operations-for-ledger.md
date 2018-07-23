---
title: Operations for Ledger
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=operations&endpoint=for_ledger
---

This endpoint returns all [operations](../resources/operation.md) that occurred in a given [ledger](../resources/ledger.md).

## Request

```
GET /ledgers/{id}/operations{?cursor,limit,order}
```

### Arguments

| name     | notes                          | description                                                      | example      |
| ------   | -------                        | -----------                                                      | -------      |
| `id`     | required, number               | Ledger ID                                                        | `6985922`    |
| `?cursor`| optional, default _null_       | A paging token, specifying where to start returning records from.| `12884905984`|
| `?order` | optional, string, default `asc`| The order in which to return rows, "asc" or "desc".              | `asc`        |
| `?limit` | optional, number, default `10` | Maximum number of records to return.                             | `200`        |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/ledgers/6985922/operations?limit=1"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.operations()
  .forLedger("6985922")
  .call()
  .then(function (operationsResult) {
    console.log(operationsResult.records);
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a list of operations in a given ledger.  See [operation resource](../resources/operation.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/ledgers/6985922/operations?cursor=\u0026limit=1\u0026order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/ledgers/6985922/operations?cursor=30004306522411009\u0026limit=1\u0026order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/ledgers/6985922/operations?cursor=30004306522411009\u0026limit=1\u0026order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://horizon-testnet.stellar.org/operations/30004306522411009"
          },
          "transaction": {
            "href": "https://horizon-testnet.stellar.org/transactions/ca375ca4c5f084c9654459fd5c84cfa58c627adad6a91ffb848585a7989be044"
          },
          "effects": {
            "href": "https://horizon-testnet.stellar.org/operations/30004306522411009/effects"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc\u0026cursor=30004306522411009"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc\u0026cursor=30004306522411009"
          }
        },
        "id": "30004306522411009",
        "paging_token": "30004306522411009",
        "source_account": "GBLMPWYW2R7ICYRM62GSLW5B567ETK5O64N34VPT4RWH6PMWPMIG3FHT",
        "type": "manage_offer",
        "type_i": 3,
        "created_at": "2018-01-29T02:39:49Z",
        "transaction_hash": "ca375ca4c5f084c9654459fd5c84cfa58c627adad6a91ffb848585a7989be044",
        "amount": "0.0999454",
        "price": "19776.0950519",
        "price_r": {
          "n": 703020403,
          "d": 35549
        },
        "buying_asset_type": "native",
        "selling_asset_type": "credit_alphanum4",
        "selling_asset_code": "BTC",
        "selling_asset_issuer": "GCNSGHUCG5VMGLT5RIYYZSO7VQULQKAJ62QA33DBC5PPBSO57LFWVV6P",
        "offer_id": 81843
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no ledger whose ID matches the `id` argument.
