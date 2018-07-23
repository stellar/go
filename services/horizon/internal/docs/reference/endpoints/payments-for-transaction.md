---
title: Payments for Transaction
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=payments&endpoint=for_transaction
---

This endpoint represents all payment-related [operations](../resources/operation.md) that are part of a given [transaction](../resources/transaction.md).

The operations that can be returned in by this endpoint are:
- `create_account`
- `payment`
- `path_payment`
- `account_merge`

## Request

```
GET /transactions/{hash}/payments{?cursor,limit,order}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `hash` | required, string | A transaction hash, hex-encoded | `6391dd190f15f7d1665ba53c63842e368f485651a53d8d852ed442a446d1c69a` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `12884905984` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc".               | `asc`         |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/transactions/622d566df3ee61848be3c3211f614310082f4a09c959cbee7be6990b0dccdba8/payments"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.payments()
  .forTransaction("622d566df3ee61848be3c3211f614310082f4a09c959cbee7be6990b0dccdba8")
  .call()
  .then(function (paymentResult) {
    console.log(paymentResult.records);
  })
  .catch(function (err) {
    console.log(err);
  })
```

## Response

This endpoint responds with a list of payments operations that are part of a given transaction. See [operation resource](../resources/operation.md) for more information about operations (and payment operations).

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/transactions/622d566df3ee61848be3c3211f614310082f4a09c959cbee7be6990b0dccdba8/payments?cursor=\u0026limit=10\u0026order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/transactions/622d566df3ee61848be3c3211f614310082f4a09c959cbee7be6990b0dccdba8/payments?cursor=43456504870744065\u0026limit=10\u0026order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/transactions/622d566df3ee61848be3c3211f614310082f4a09c959cbee7be6990b0dccdba8/payments?cursor=43456504870744065\u0026limit=10\u0026order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://horizon-testnet.stellar.org/operations/43456504870744065"
          },
          "transaction": {
            "href": "https://horizon-testnet.stellar.org/transactions/622d566df3ee61848be3c3211f614310082f4a09c959cbee7be6990b0dccdba8"
          },
          "effects": {
            "href": "https://horizon-testnet.stellar.org/operations/43456504870744065/effects"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/effects?order=desc\u0026cursor=43456504870744065"
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/effects?order=asc\u0026cursor=43456504870744065"
          }
        },
        "id": "43456504870744065",
        "paging_token": "43456504870744065",
        "source_account": "GCEDON5M2HFQ5P5MVWYGKH3K4S3ZKQFZ5AYLWNXPSGVOHLJUXQRYL35Z",
        "type": "payment",
        "type_i": 1,
        "created_at": "2018-07-20T16:12:46Z",
        "transaction_hash": "622d566df3ee61848be3c3211f614310082f4a09c959cbee7be6990b0dccdba8",
        "asset_type": "credit_alphanum12",
        "asset_code": "nCntGameCoin",
        "asset_issuer": "GDLMDXI6EVVUIXWRU4S2YVZRMELHUEX3WKOX6XFW77QQC6KZJ4CZ7NRB",
        "from": "GCEDON5M2HFQ5P5MVWYGKH3K4S3ZKQFZ5AYLWNXPSGVOHLJUXQRYL35Z",
        "to": "GAWHFIIG2LTBO4Q5ZCTMPGYVV3FSQE4TCXPFTL44V5BN5FLR3GRGJB5G",
        "amount": "1.0000000"
      }
    ]
  }
}
```



## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no transaction whose ID matches the `hash` argument.
