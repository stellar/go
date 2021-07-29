---
title: Effects for Operation
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=effects&endpoint=for_operation
---

This endpoint represents all [effects](../resources/effect.md) that occurred as a result of a given [operation](../resources/operation.md).

## Request

```
GET /operations/{id}/effects{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `id` | required, number | An operation ID | `1099511631873` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit` | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/operations/1099511631873/effects"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.effects()
  .forOperation("1099511631873")
  .call()
  .then(function (effectResults) {
    // page 1
    console.log(JSON.stringify(effectResults.records))
  })
  .catch(function (err) {
    console.log(err)
  })

```

## Response

This endpoint responds with a list of effects on the ledger as a result of a given operation. See [effect resource](../resources/effect.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631873-1"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631873-1"
      }
    },
    "id": "0000001099511631873-0000000001",
    "paging_token": "1099511631873-1",
    "account": "GDE3XSDA4G7MZJXZ6SYYD7CHQSOUFMEDTSU2WINVJ42DOFOCBTLGI5O4",
    "type": "account_created",
    "type_i": 0,
    "created_at": "2021-04-13T13:55:32Z",
    "starting_balance": "101.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631873-2"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631873-2"
      }
    },
    "id": "0000001099511631873-0000000002",
    "paging_token": "1099511631873-2",
    "account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "account_debited",
    "type_i": 3,
    "created_at": "2021-04-13T13:55:32Z",
    "asset_type": "native",
    "amount": "101.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631873-3"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631873-3"
      }
    },
    "id": "0000001099511631873-0000000003",
    "paging_token": "1099511631873-3",
    "account": "GDE3XSDA4G7MZJXZ6SYYD7CHQSOUFMEDTSU2WINVJ42DOFOCBTLGI5O4",
    "type": "signer_created",
    "type_i": 10,
    "created_at": "2021-04-13T13:55:32Z",
    "weight": 1,
    "public_key": "GDE3XSDA4G7MZJXZ6SYYD7CHQSOUFMEDTSU2WINVJ42DOFOCBTLGI5O4",
    "key": ""
  }
]
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` errors will be returned if there are no effects for operation whose ID matches the `id` argument.
