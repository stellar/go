---
title: Offer Details
---

Returns information and links relating to a single [offer](../resources/offer.md).

## Request

```
GET /offers/{offer}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `offer` | required, string | Offer ID | `1` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/offers/1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.offers('accounts', 'GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY')
  .call()
  .then(function (offerResult) {
    console.log(JSON.stringify(offerResult));
  })
  .catch(function (err) {
    console.error(err);
  })
```

## Response

This endpoint responds with the details of a single offer for a given ID. See [offer resource](../resources/offer.md) for reference.

### Example Response

```json
{
  "records": [
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/offers/1"
        },
        "offer_maker": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY"
        }
      },
      "id": "1",
      "paging_token": "1",
      "seller": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
      "selling": {
        "asset_type": "credit_alphanum4",
        "asset_code": "USD",
        "asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ"
      },
      "buying": {
        "asset_type": "native"
      },
      "amount": "1.0000000",
      "price_r": {
        "n": 2,
        "d": 1
      },
      "price": "2.0000000",
      "last_modified_ledger": 963496,
      "last_modified_time": "2021-06-16T18:15:07Z"
    }
  ]
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no offer whose ID matches the `offer` argument.
