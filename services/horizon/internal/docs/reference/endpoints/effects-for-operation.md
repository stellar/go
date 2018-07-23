---
title: Effects for Operation
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=effects&endpoint=for_operation
---

This endpoint represents all [effects](../resources/effect.md) that occurred as a result of a given [operation](../resources/operation.md).

## Request

```
GET /operations/{id}/effects{?cursor,limit,order}
```

### Arguments

| name     | notes                          | description                                                      | example      |
| ------   | -------                        | -----------                                                      | -------      |
| `id`     | required, number               | An operation ID                                                  | `10157597659137`|
| `?cursor`| optional, default _null_       | A paging token, specifying where to start returning records from.| `12884905984`|
| `?order` | optional, string, default `asc`| The order in which to return rows, "asc" or "desc".              | `asc`        |
| `?limit` | optional, number, default `10` | Maximum number of records to return.                             | `200`        |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/operations/10157597659137/effects"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.effects()
  .forOperation("10157597659137")
  .call()
  .then(function (effectResults) {
    //page 1
    console.log(effectResults.records)
  })
  .catch(function (err) {
    console.log(err)
  })

```

## Response

This endpoint responds with a list of effects on the ledger as a result of a given operation. See [effect resource](../resources/effect.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/operations/10157597659137/effects?cursor=&limit=10&order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/operations/10157597659137/effects?cursor=10157597659137-3&limit=10&order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/operations/10157597659137/effects?cursor=10157597659137-1&limit=10&order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/10157597659137"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc&cursor=10157597659137-1"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc&cursor=10157597659137-1"
          }
        },
        "id": "0000010157597659137-0000000001",
        "paging_token": "10157597659137-1",
        "account": "GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K",
        "type": "account_created",
        "type_i": 0,
        "created_at": "2017-03-20T19:50:52Z",
        "starting_balance": "50000000.0000000"
      },
      {
        "_links": {
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/10157597659137"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc&cursor=10157597659137-2"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc&cursor=10157597659137-2"
          }
        },
        "id": "0000010157597659137-0000000002",
        "paging_token": "10157597659137-2",
        "account": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
        "type": "account_debited",
        "type_i": 3,
        "created_at": "2017-03-20T19:50:52Z",
        "asset_type": "native",
        "amount": "50000000.0000000"
      },
      {
        "_links": {
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/10157597659137"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc&cursor=10157597659137-3"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc&cursor=10157597659137-3"
          }
        },
        "id": "0000010157597659137-0000000003",
        "paging_token": "10157597659137-3",
        "account": "GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K",
        "type": "signer_created",
        "type_i": 10,
        "created_at": "2017-03-20T19:50:52Z",
        "weight": 1,
        "public_key": "GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K",
        "key": ""
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` errors will be returned if there are no effects for operation whose ID matches the `id` argument.
