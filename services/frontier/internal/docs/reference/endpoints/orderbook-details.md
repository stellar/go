---
title: Orderbook Details
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=order_book&endpoint=details
---

People on the DigitalBits network can make [offers](../resources/offer.md) to buy or sell assets.
These offers are summarized by the assets being bought and sold in
[orderbooks](../resources/orderbook.md).

Frontier will return, for each orderbook, a summary of the orderbook and the bids and asks
associated with that orderbook.

This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to
listen as offers are processed in the DigitalBits network.  If called in streaming mode Frontier will
start at the earliest known offer unless a `cursor` is set. In that case it will start from the
`cursor`. You can also set `cursor` value to `now` to only stream offers created since your request
time.

## Request

```
GET /order_book?selling_asset_type={selling_asset_type}&selling_asset_code={selling_asset_code}&selling_asset_issuer={selling_asset_issuer}&buying_asset_type={buying_asset_type}&buying_asset_code={buying_asset_code}&buying_asset_issuer={buying_asset_issuer}&limit={limit}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `selling_asset_type` | required, string | Type of the Asset being sold | `native` |
| `selling_asset_code` | optional, string | Code of the Asset being sold | `USD` |
| `selling_asset_issuer` | optional, string | Account ID of the issuer of the Asset being sold | `GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC` |
| `buying_asset_type` | required, string | Type of the Asset being bought | `credit_alphanum4` |
| `buying_asset_code` | optional, string | Code of the Asset being bought | `BTC` |
| `buying_asset_issuer` | optional, string | Account ID of the issuer of the Asset being bought | `GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ` |
| `limit` | optional, string | Limit the number of items returned | `20` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/order_book?selling_asset_type=credit_alphanum4&selling_asset_code=EUR&selling_asset_issuer=GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC&buying_asset_type=credit_alphanum4&buying_asset_code=USD&buying_asset_issuer=GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ&limit=20"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.orderbook(new DigitalBitsSdk.Asset('EUR', 'GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC'), new DigitalBitsSdk.Asset('USD', 'GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ'))
  .call()
  .then(function(resp) {
    console.log(JSON.stringify(resp));
  })
  .catch(function(err) {
    console.log(err);
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var orderbookHandler = function (orderbookResponse) {
  console.log(orderbookResponse);
};

var es = server.orderbook(new DigitalBitsSdk.Asset('EUR', 'GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC'), new DigitalBitsSdk.Asset('USD', 'GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ'))
  .cursor('now')
  .stream({
    onmessage: orderbookHandler
  })
```

## Response

The summary of the orderbook and its bids and asks.

## Example Response
```json
{
  "bids": [
    {
      "price_r": {
        "n": 2,
        "d": 1
      },
      "price": "2.0000000",
      "amount": "2.0000000"
    }
  ],
  "asks": [],
  "base": {
    "asset_type": "credit_alphanum4",
    "asset_code": "EUR",
    "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC"
  },
  "counter": {
    "asset_type": "credit_alphanum4",
    "asset_code": "USD",
    "asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ"
  }
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
