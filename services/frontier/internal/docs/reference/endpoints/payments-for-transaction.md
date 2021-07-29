---
title: Payments for Transaction
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=payments&endpoint=for_transaction
---

This endpoint represents all payment-related [operations](../resources/operation.md) that are part
of a given [transaction](../resources/transaction.md).

The operations that can be returned in by this endpoint are:

- `create_account`
- `payment`
- `path_payment`
- `account_merge`

### Warning - failed transactions

"Payments for Transaction" endpoint returns list of payments of successful or failed transactions
(that are also included in DigitalBits ledger). Always check the payment status in this endpoint using
`transaction_successful` field!

## Request

```
GET /transactions/{hash}/payments{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `hash` | required, string | A transaction hash, hex-encoded, lowercase. | `c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit` | optional, number, default `10` | Maximum number of records to return. | `200` |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the payments in the response. | `transactions` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f/payments"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.payments()
  .forTransaction("c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f")
  .call()
  .then(function (paymentResult) {
    console.log(JSON.stringify(paymentResult.records));
  })
  .catch(function (err) {
    console.log(err);
  })
```

## Response

This endpoint responds with a list of payments operations that are part of a given transaction. See
[operation resource](../resources/operation.md) for more information about operations (and payment
operations).

### Example Response

```json
[
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
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no
  transaction whose ID matches the `hash` argument.
