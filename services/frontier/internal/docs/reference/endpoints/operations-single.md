---
title: Operation Details
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=operations&endpoint=single
---

The operation details endpoint provides information on a single
[operation](../resources/operation.md). The operation ID provided in the `id` argument specifies
which operation to load.

### Warning - failed transactions

Operations can be part of successful or failed transactions (failed transactions are also included
in DigitalBits ledger). Always check operation status using `transaction_successful` field!

## Request

```
GET /operations/{id}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `id` | required, number | An operation ID. | 1099511631873 |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the operations in the response. | `transactions` |

### curl Example Request

```sh
curl https://frontier.testnet.digitalbits.io/operations/1099511631873
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.operations()
  .operation('1099511631873')
  .call()
  .then(function (operationsResult) {
    console.log(JSON.stringify(operationsResult))
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a single Operation.  See [operation resource](../resources/operation.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631873"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631873"
    }
  },
  "id": "1099511631873",
  "paging_token": "1099511631873",
  "transaction_successful": true,
  "source_account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "type": "create_account",
  "type_i": 0,
  "created_at": "2021-04-13T13:55:32Z",
  "transaction_hash": "081c8114fe004413a325294413c9372ce47ac4fc6925b5b994d80f854e0bddf9",
  "starting_balance": "101.0000000",
  "funder": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
  "account": "GDE3XSDA4G7MZJXZ6SYYD7CHQSOUFMEDTSU2WINVJ42DOFOCBTLGI5O4"
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if the
  there is no operation that matches the ID argument, i.e. the operation does not exist.
