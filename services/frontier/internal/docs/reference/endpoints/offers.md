---
title: Offers
---

People on the DigitalBits network can make [offers](../resources/offer.md) to buy or sell assets. This
endpoint represents all the current offers, allowing filtering by `seller`, `selling_asset` or `buying_asset`.

## Request

```
GET /offers{?selling_asset_type,selling_asset_issuer,selling_asset_code,buying_asset_type,buying_asset_issuer,buying_asset_code,seller,cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?seller` | optional, string | Account ID of the offer creator  | `GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY` |
| `?selling` | optional, string | Asset being sold | `native` or `EUR:GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC` |
| `?buying` | optional, string | Asset being bought | `native` or `USD:GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ` |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/offers{?selling_asset_type,selling_asset_issuer,selling_asset_code,buying_asset_type,buying_asset_issuer,buying_asset_code,seller,cursor,limit,order}"
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

### Example Response

```json
{
  "records": [
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/offers/8"
        },
        "offer_maker": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M"
        }
      },
      "id": "8",
      "paging_token": "8",
      "seller": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
      "selling": {
        "asset_type": "credit_alphanum4",
        "asset_code": "UAH",
        "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
      },
      "buying": {
        "asset_type": "credit_alphanum4",
        "asset_code": "USD",
        "asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ"
      },
      "amount": "1000.0000000",
      "price_r": {
        "n": 1,
        "d": 1
      },
      "price": "1.0000000",
      "last_modified_ledger": 973175,
      "last_modified_time": "2021-06-17T09:42:39Z"
    },
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/offers/9"
        },
        "offer_maker": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCSYKECRGY6VEF4F4KBZEEPXLYDLUGNZFCCXWR7SNRADN3NYYK67GQKF"
        }
      },
      "id": "9",
      "paging_token": "9",
      "seller": "GCSYKECRGY6VEF4F4KBZEEPXLYDLUGNZFCCXWR7SNRADN3NYYK67GQKF",
      "selling": {
        "asset_type": "credit_alphanum4",
        "asset_code": "EUR",
        "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC"
      },
      "buying": {
        "asset_type": "credit_alphanum4",
        "asset_code": "UAH",
        "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
      },
      "amount": "1000.0000000",
      "price_r": {
        "n": 1,
        "d": 1
      },
      "price": "1.0000000",
      "last_modified_ledger": 973254,
      "last_modified_time": "2021-06-17T09:50:13Z"
    }
  ]
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
