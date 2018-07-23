---
title: Effects for Transaction
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=effects&endpoint=for_transaction
---

This endpoint represents all [effects](../resources/effect.md) that occurred as a result of a given [transaction](../resources/transaction.md).

## Request

```
GET /transactions/{hash}/effects{?cursor,limit,order}
```

## Arguments

| name     | notes                          | description                                                      | example                                                           |
| ------   | -------                        | -----------                                                      | -------                                                           |
| `hash`   | required, string               | A transaction hash, hex-encoded                                  | `17a670bc424ff5ce3b386dbfaae9990b66a2a37b4fbe51547e8794962a3f9e6a`|
| `?cursor`| optional, default _null_       | A paging token, specifying where to start returning records from.| `12884905984`                                                     |
| `?order` | optional, string, default `asc`| The order in which to return rows, "asc" or "desc".              | `asc`                                                             |
| `?limit` | optional, number, default `10` | Maximum number of records to return.                             | `200`                                                             |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/transactions/17a670bc424ff5ce3b386dbfaae9990b66a2a37b4fbe51547e8794962a3f9e6a/effects?limit=1"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.effects()
  .forTransaction("17a670bc424ff5ce3b386dbfaae9990b66a2a37b4fbe51547e8794962a3f9e6a")
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

This endpoint responds with a list of effects on the ledger as a result of a given transaction. See [effect resource](../resources/effect.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/transactions/17a670bc424ff5ce3b386dbfaae9990b66a2a37b4fbe51547e8794962a3f9e6a/effects?cursor=&limit=1&order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/transactions/17a670bc424ff5ce3b386dbfaae9990b66a2a37b4fbe51547e8794962a3f9e6a/effects?cursor=10157597659137-1&limit=1&order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/transactions/17a670bc424ff5ce3b386dbfaae9990b66a2a37b4fbe51547e8794962a3f9e6a/effects?cursor=10157597659137-1&limit=1&order=desc"
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
      }
    ]
  }
}
```

## Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there are no effects for transaction whose hash matches the `hash` argument.
