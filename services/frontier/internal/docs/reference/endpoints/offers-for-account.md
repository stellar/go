---
title: Offers for Account
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=offers&endpoint=for_account
---


People on the DigitalBits network can make [offers](../resources/offer.md) to buy or sell assets. This
endpoint represents all the offers a particular account makes.

This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to
listen as offers are processed in the DigitalBits network. If called in streaming mode Frontier will
start at the earliest known offer unless a `cursor` is set. In that case it will start from the
`cursor`. You can also set `cursor` value to `now` to only stream offers created since your request
time.

## Request

```
GET /accounts/{account}/offers{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account` | required, string | Account ID | `GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY` |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/offers"
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

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var offerHandler = function (offerResponse) {
  console.log(offerResponse);
};

var es = server.offers('accounts', 'GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY')
  .cursor('now')
  .stream({
    onmessage: offerHandler
  })
```

## Response

The list of offers.

**Note:** a response of 200 with an empty records array may either mean there are no offers for
`account_id` or `account_id` does not exist.

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
