---
title: Effects for Ledger
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=effects&endpoint=for_ledger
---

Effects are the specific ways that the ledger was changed by any operation.

This endpoint represents all [effects](../resources/effect.md) that occurred in the given [ledger](../resources/ledger.md).

## Request

```
GET /ledgers/{id}/effects{?cursor,limit,order}
```

## Arguments

| name     | notes                          | description                                                      | example      |
| ------   | -------                        | -----------                                                      | -------      |
| `id`     | required, number               | Ledger ID                                                        | `10085921`   |
| `?cursor`| optional, default _null_       | A paging token, specifying where to start returning records from.| `12884905984`|
| `?order` | optional, string, default `asc`| The order in which to return rows, "asc" or "desc".              | `asc`        |
| `?limit` | optional, number, default `10` | Maximum number of records to return.                             | `200`        |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/ledgers/10085921/effects?limit=1"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.effects()
  .forLedger("10085921")
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

This endpoint responds with a list of effects that occurred in the ledger. See [effect resource](../resources/effect.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/ledgers/10085921/effects?cursor=\u0026limit=1\u0026order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/ledgers/10085921/effects?cursor=43318700845043713-1\u0026limit=1\u0026order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/ledgers/10085921/effects?cursor=43318700845043713-1\u0026limit=1\u0026order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "operation": {
            "href": "https://horizon-testnet.stellar.org/operations/43318700845043713"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc\u0026cursor=43318700845043713-1"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc\u0026cursor=43318700845043713-1"
          }
        },
        "id": "0043318700845043713-0000000001",
        "paging_token": "43318700845043713-1",
        "account": "GB4OMVGHXSZRVT6KTXV22LPWIVYZES7B43RQ3P54BKIH3U36GOPS645Y",
        "type": "account_credited",
        "type_i": 2,
        "created_at": "2018-07-18T19:39:00Z",
        "asset_type": "credit_alphanum12",
        "asset_code": "nCntGameCoin",
        "asset_issuer": "GDLMDXI6EVVUIXWRU4S2YVZRMELHUEX3WKOX6XFW77QQC6KZJ4CZ7NRB",
        "amount": "1.0000000"
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there are no effects for a given ledger.
