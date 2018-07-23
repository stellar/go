---
title: Transaction Details
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=transactions&endpoint=single
---

The transaction details endpoint provides information on a single [transaction](../resources/transaction.md). The transaction hash provided in the `hash` argument specifies which transaction to load.

## Request

```
GET /transactions/{hash}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `hash` | required, string | A transaction hash, hex-encoded. | 85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/transactions/85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.transactions()
  .transaction("85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c")
  .call()
  .then(function (transactionResult) {
    console.log(transactionResult)
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a single Transaction.  See [transaction resource](../resources/transaction.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/transactions/85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c"
    },
    "account": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBALEQDXK5F77ZSQEQG6QOOWJVDVVWKOZN2IAKALKXIS34DA5K6OAQW7"
    },
    "ledger": {
      "href": "https://horizon-testnet.stellar.org/ledgers/10106016"
    },
    "operations": {
      "href": "https://horizon-testnet.stellar.org/transactions/85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c/operations{?cursor,limit,order}",
      "templated": true
    },
    "effects": {
      "href": "https://horizon-testnet.stellar.org/transactions/85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c/effects{?cursor,limit,order}",
      "templated": true
    },
    "precedes": {
      "href": "https://horizon-testnet.stellar.org/transactions?order=asc&cursor=43405008212856832"
    },
    "succeeds": {
      "href": "https://horizon-testnet.stellar.org/transactions?order=desc&cursor=43405008212856832"
    }
  },
  "id": "85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c",
  "paging_token": "43405008212856832",
  "hash": "85b0084a82a055dc1b3c9312277c97b13baca5377d23d4b4fde36c01467edf9c",
  "ledger": 10106016,
  "created_at": "2018-07-19T23:33:35Z",
  "source_account": "GBALEQDXK5F77ZSQEQG6QOOWJVDVVWKOZN2IAKALKXIS34DA5K6OAQW7",
  "source_account_sequence": "37676552632212983",
  "fee_paid": 100,
  "operation_count": 1,
  "envelope_xdr": "AAAAAECyQHdXS//mUCQN6DnWTUda2U7LdIAoC1XRLfBg6rzgAAAAZACF2qAAAR33AAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAeOZUx7yzGs/KneutLfZFcZJL4ebjDb+8CpB9034zny8AAAACbkNudEdhbWVDb2luAAAAANbB3R4la0Re0aclrFcxYRZ6EvuynX9ctv/hAXlZTwWfAAAAAACYloAAAAAAAAAAAWDqvOAAAABAw/3HdgLG9+9ScBjSDi0htbmOruoaZaQKbCrVIaucQde7zJmexofgt+nnSzx+Sr2+uiEugt0J5hnIQq1kICitCg==",
  "result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
  "result_meta_xdr": "AAAAAAAAAAEAAAAEAAAAAwCaNJQAAAABAAAAAECyQHdXS//mUCQN6DnWTUda2U7LdIAoC1XRLfBg6rzgAAAAAm5DbnRHYW1lQ29pbgAAAADWwd0eJWtEXtGnJaxXMWEWehL7sp1/XLb/4QF5WU8FnwAABsf9bG0Af/////////8AAAABAAAAAAAAAAAAAAABAJo0oAAAAAEAAAAAQLJAd1dL/+ZQJA3oOdZNR1rZTst0gCgLVdEt8GDqvOAAAAACbkNudEdhbWVDb2luAAAAANbB3R4la0Re0aclrFcxYRZ6EvuynX9ctv/hAXlZTwWfAAAGx/zT1oB//////////wAAAAEAAAAAAAAAAAAAAAMAmjSfAAAAAQAAAAB45lTHvLMaz8qd660t9kVxkkvh5uMNv7wKkH3TfjOfLwAAAAJuQ250R2FtZUNvaW4AAAAA1sHdHiVrRF7RpyWsVzFhFnoS+7Kdf1y2/+EBeVlPBZ8AAAbSqRq6gH//////////AAAAAQAAAAAAAAAAAAAAAQCaNKAAAAABAAAAAHjmVMe8sxrPyp3rrS32RXGSS+Hm4w2/vAqQfdN+M58vAAAAAm5DbnRHYW1lQ29pbgAAAADWwd0eJWtEXtGnJaxXMWEWehL7sp1/XLb/4QF5WU8FnwAABtKps1EAf/////////8AAAABAAAAAAAAAAA=",
  "fee_meta_xdr": "AAAAAgAAAAMAmjSIAAAAAAAAAABAskB3V0v/5lAkDeg51k1HWtlOy3SAKAtV0S3wYOq84AAAABdIBzPoAIXaoAABHfYAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAmjSgAAAAAAAAAABAskB3V0v/5lAkDeg51k1HWtlOy3SAKAtV0S3wYOq84AAAABdIBzOEAIXaoAABHfcAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
  "memo_type": "none",
  "signatures": [
    "w/3HdgLG9+9ScBjSDi0htbmOruoaZaQKbCrVIaucQde7zJmexofgt+nnSzx+Sr2+uiEugt0J5hnIQq1kICitCg=="
  ]
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no transaction whose ID matches the `hash` argument.
