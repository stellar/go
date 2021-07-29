---
title: Payments for Ledger
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=payments&endpoint=for_ledger
---

This endpoint represents all payment-related [operations](../resources/operation.md) that are part
of a valid [transactions](../resources/transaction.md) in a given [ledger](../resources/ledger.md).

The operations that can be returned in by this endpoint are:

- `create_account`
- `payment`
- `path_payment`
- `account_merge`

## Request

```
GET /ledgers/{id}/payments{?cursor,limit,order,include_failed}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `id` | required, number | Ledger ID | `957773` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include payments of failed transactions in results. | `true` |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the payments in the response. | `transactions` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/ledgers/957773/payments?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.payments()
  .forLedger("957773")
  .call()
  .then(function (paymentResult) {
    console.log(JSON.stringify(paymentResult))
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a list of payment operations in a given ledger. See [operation
resource](../resources/operation.md) for more information about operations (and payment
operations).

### Example Response

```json
{
  "records": [
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/operations/4114853547479041"
        },
        "transaction": {
          "href": "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f"
        },
        "effects": {
          "href": "https://frontier.testnet.digitalbits.io/operations/4114853547479041/effects"
        },
        "succeeds": {
          "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4114853547479041"
        },
        "precedes": {
          "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4114853547479041"
        }
      },
      "id": "4114853547479041",
      "paging_token": "4114853547479041",
      "transaction_successful": true,
      "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
      "type": "payment",
      "type_i": 1,
      "created_at": "2021-06-16T09:36:04Z",
      "transaction_hash": "c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f",
      "asset_type": "credit_alphanum4",
      "asset_code": "HUF",
      "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
      "from": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
      "to": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
      "amount": "50000.0000000"
    }
  ]
}
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no ledger whose ID matches the `id` argument.
