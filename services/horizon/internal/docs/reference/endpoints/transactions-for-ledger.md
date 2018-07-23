---
title: Transactions for Ledger
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=transactions&endpoint=for_ledger
---

This endpoint represents all [transactions](../resources/transaction.md) in a given [ledger](../resources/ledger.md).

## Request

```
GET /ledgers/{id}/transactions{?cursor,limit,order}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `id` | required, number | Ledger ID | `9000000` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `12884905984` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/ledgers/9000000/transactions?limit=1"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.transactions()
  .forLedger("9000000")
  .limit("1")
  .call()
  .then(function (accountResults) {
    console.log(accountResults.records)
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a list of transactions in a given ledger.  See [transaction resource](../resources/transaction.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/ledgers/9000000/transactions?cursor=\u0026limit=1\u0026order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/ledgers/9000000/transactions?cursor=38654705664004096\u0026limit=1\u0026order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/ledgers/9000000/transactions?cursor=38654705664004096\u0026limit=1\u0026order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://horizon-testnet.stellar.org/transactions/2b5742600b45f2051d9e4d85d900e271d11c46ec1fccfb54469b1e5d3326585e"
          },
          "account": {
            "href": "https://horizon-testnet.stellar.org/accounts/GBBPUGFZQCQX4MCU5SJZECQZCAISVI6EQXQSW2MNJXPCK3QVSWGYYAA7"
          },
          "ledger": {
            "href": "https://horizon-testnet.stellar.org/ledgers/9000000"
          },
          "operations": {
            "href": "https://horizon-testnet.stellar.org/transactions/2b5742600b45f2051d9e4d85d900e271d11c46ec1fccfb54469b1e5d3326585e/operations{?cursor,limit,order}",
            "templated": true
          },
          "effects": {
            "href": "https://horizon-testnet.stellar.org/transactions/2b5742600b45f2051d9e4d85d900e271d11c46ec1fccfb54469b1e5d3326585e/effects{?cursor,limit,order}",
            "templated": true
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/transactions?order=asc\u0026cursor=38654705664004096"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/transactions?order=desc\u0026cursor=38654705664004096"
          }
        },
        "id": "2b5742600b45f2051d9e4d85d900e271d11c46ec1fccfb54469b1e5d3326585e",
        "paging_token": "38654705664004096",
        "hash": "2b5742600b45f2051d9e4d85d900e271d11c46ec1fccfb54469b1e5d3326585e",
        "ledger": 9000000,
        "created_at": "2018-05-16T20:39:57Z",
        "source_account": "GBBPUGFZQCQX4MCU5SJZECQZCAISVI6EQXQSW2MNJXPCK3QVSWGYYAA7",
        "source_account_sequence": "26088580543780806",
        "fee_paid": 100,
        "operation_count": 1,
        "envelope_xdr": "AAAAAEL6GLmAoX4wVOyTkgoZEBEqo8SF4StpjU3eJW4VlY2MAAAAZABcr20AAZfGAAAAAAAAAAEAAAAUUDowLjc1MTA4NjY5OTkyODg2MjYAAAABAAAAAAAAAAEAAAAAOTWBGXr2eMo9a57pGIn+2nsIC5m6m9n68vTStJnCP7wAAAACWmlmQ29pbgAAAAAAAAAAAEL6GLmAoX4wVOyTkgoZEBEqo8SF4StpjU3eJW4VlY2MAAAAAAABhqAAAAAAAAAAARWVjYwAAABA4yDYtVvMiSuHkh0syb43MedFwsjHgqmFtA618Z5e/dZ3zobWMY3SPvY81mNw8fbSpHLg0lYzIjHNo5dbAvCaAw==",
        "result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
        "result_meta_xdr": "AAAAAAAAAAEAAAACAAAAAwCJVD4AAAABAAAAADk1gRl69njKPWue6RiJ/tp7CAuZupvZ+vL00rSZwj+8AAAAAlppZkNvaW4AAAAAAAAAAABC+hi5gKF+MFTsk5IKGRARKqPEheEraY1N3iVuFZWNjAAAAAACAjigf/////////8AAAABAAAAAAAAAAAAAAABAIlUQAAAAAEAAAAAOTWBGXr2eMo9a57pGIn+2nsIC5m6m9n68vTStJnCP7wAAAACWmlmQ29pbgAAAAAAAAAAAEL6GLmAoX4wVOyTkgoZEBEqo8SF4StpjU3eJW4VlY2MAAAAAAIDv0B//////////wAAAAEAAAAAAAAAAA==",
        "fee_meta_xdr": "AAAAAgAAAAMAiVQ+AAAAAAAAAABC+hi5gKF+MFTsk5IKGRARKqPEheEraY1N3iVuFZWNjAAADSzkRrSMAFyvbQABl8UAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAiVRAAAAAAAAAAABC+hi5gKF+MFTsk5IKGRARKqPEheEraY1N3iVuFZWNjAAADSzkRrQoAFyvbQABl8YAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
        "memo_type": "text",
        "memo": "P:0.7510866999288626",
        "signatures": [
          "4yDYtVvMiSuHkh0syb43MedFwsjHgqmFtA618Z5e/dZ3zobWMY3SPvY81mNw8fbSpHLg0lYzIjHNo5dbAvCaAw=="
        ]
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no ledgers whose sequence matches the `id` argument.
