---
title: Payments for Ledger
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=payments&endpoint=for_ledger
---

This endpoint represents all payment-releated [operations](../resources/operation.md) that are part of a valid [transactions](../resources/transaction.md) in a given [ledger](../resources/ledger.md).

The operations that can be returned in by this endpoint are:
- `create_account`
- `payment`
- `path_payment`
- `account_merge`

## Request

```
GET /ledgers/{id}/payments{?cursor,limit,order}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `id` | required, number | Ledger ID | `10009866` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `12884905984` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/ledgers/10009866/payments?limit=1"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk')
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.payments()
  .forLedger("10009866")
  .call()
  .then(function (paymentResult) {
    console.log(paymentResult)
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a list of payment operations in a given ledger.  See [operation resource](../resources/operation.md) for more information about operations (and payment operations).

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/ledgers/10009866/payments?cursor=\u0026limit=1\u0026order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/ledgers/10009866/payments?cursor=42992047107346433\u0026limit=1\u0026order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/ledgers/10009866/payments?cursor=42992047107346433\u0026limit=1\u0026order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://horizon-testnet.stellar.org/operations/42992047107346433"
          },
          "transaction": {
            "href": "https://horizon-testnet.stellar.org/transactions/e235cf90e6cc5c84ec6912dc3cbd13b627f43eca8ac59eae34872c224c5a728c"
          },
          "effects": {
            "href": "https://horizon-testnet.stellar.org/operations/42992047107346433/effects"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc\u0026cursor=42992047107346433"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc\u0026cursor=42992047107346433"
          }
        },
        "id": "42992047107346433",
        "paging_token": "42992047107346433",
        "source_account": "GCANEDZUIL7MBBWWCTQ25534UMCOIQLEH3IPOCU7W7H6LK7I35WAE7BI",
        "type": "payment",
        "type_i": 1,
        "created_at": "2018-07-14T09:59:20Z",
        "transaction_hash": "e235cf90e6cc5c84ec6912dc3cbd13b627f43eca8ac59eae34872c224c5a728c",
        "asset_type": "credit_alphanum12",
        "asset_code": "nCntGameCoin",
        "asset_issuer": "GDLMDXI6EVVUIXWRU4S2YVZRMELHUEX3WKOX6XFW77QQC6KZJ4CZ7NRB",
        "from": "GCANEDZUIL7MBBWWCTQ25534UMCOIQLEH3IPOCU7W7H6LK7I35WAE7BI",
        "to": "GC25MF2YFV2KTBVVVL7HT3PHAMGV7N46DVL75MJU4IVXSXVTAOIIHKCM",
        "amount": "1.0000000"
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no ledger whose ID matches the `id` argument.
