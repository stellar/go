---
title: Operation Details
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=operations&endpoint=single
---

The operation details endpoint provides information on a single [operation](../resources/operation.md). The operation ID provided in the `id` argument specifies which operation to load.

## Request

```
GET /operations/{id}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `id` | required, number | An operation ID. | 43685160339648551 |

### curl Example Request

```sh
curl https://horizon-testnet.stellar.org/operations/43685160339648551
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.operations()
  .operation('43685160339648551')
  .call()
  .then(function (operationsResult) {
    console.log(operationsResult)
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
      "href": "https://horizon-testnet.stellar.org/operations/43685160339648551"
    },
    "transaction": {
      "href": "https://horizon-testnet.stellar.org/transactions/3d13f059842c775096a20d7132bf4ac6c962811cb0313ff43c7c4881ee9119f3"
    },
    "effects": {
      "href": "https://horizon-testnet.stellar.org/operations/43685160339648551/effects"
    },
    "succeeds": {
      "href": "https://horizon-testnet.stellar.org/effects?order=desc\u0026cursor=43685160339648551"
    },
    "precedes": {
      "href": "https://horizon-testnet.stellar.org/effects?order=asc\u0026cursor=43685160339648551"
    }
  },
  "id": "43685160339648551",
  "paging_token": "43685160339648551",
  "source_account": "GDWGJSTUVRNFTR7STPUUHFWQYAN6KBVWCZT2YN7MY276GCSSXSWPS6JY",
  "type": "manage_offer",
  "type_i": 3,
  "created_at": "2018-07-23T18:14:46Z",
  "transaction_hash": "3d13f059842c775096a20d7132bf4ac6c962811cb0313ff43c7c4881ee9119f3",
  "amount": "828.3700000",
  "price": "0.2898700",
  "price_r": {
    "n": 28987,
    "d": 100000
  },
  "buying_asset_type": "credit_alphanum4",
  "buying_asset_code": "USD",
  "buying_asset_issuer": "GA7RXKBZOUA3FCUUS65JRLU5SZD3XBQUGRWL7NVQWU5QOXQW2LUZNBFZ",
  "selling_asset_type": "native",
  "offer_id": 526476
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no  whose ID matches the `account` argument.
