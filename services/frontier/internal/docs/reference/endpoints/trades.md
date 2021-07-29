---
title: Trades
---

People on the DigitalBits network can make [offers](../resources/offer.md) to buy or sell assets. When
an offer is fully or partially fulfilled, a [trade](../resources/trade.md) happens.

Trades can be filtered for a specific orderbook, defined by an asset pair: `base` and `counter`.

This endpoint can also be used in [streaming](../streaming.md) mode, making it possible to listen
for new trades as they occur on the DigitalBits network.

If called in streaming mode Frontier will start at the earliest known trade unless a `cursor` is
set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only
stream trades created since your request time.

## Request

```
GET /trades?base_asset_type={base_asset_type}&base_asset_code={base_asset_code}&base_asset_issuer={base_asset_issuer}&counter_asset_type={counter_asset_type}&counter_asset_code={counter_asset_code}&counter_asset_issuer={counter_asset_issuer}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `base_asset_type` | optional, string | Type of base asset | `native` |
| `base_asset_code` | optional, string | Code of base asset, not required if type is `native` | `USD` |
| `base_asset_issuer` | optional, string | Issuer of base asset, not required if type is `native` | `GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ` |
| `counter_asset_type` | optional, string | Type of counter asset  | `credit_alphanum4` |
| `counter_asset_code` | optional, string | Code of counter asset, not required if type is `native` | `EUR` |
| `counter_asset_issuer` | optional, string | Issuer of counter asset, not required if type is `native` | `GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC` |
| `offer_id` | optional, string | filter for by a specific offer id | `5` |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order, in terms of timeline, in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request
```sh
curl https://frontier.testnet.digitalbits.io/trades?base_asset_type=native&counter_asset_code=USD&counter_asset_issuer=GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ&counter_asset_type=credit_alphanum4&limit=2&order=desc
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.trades()
  .call()
  .then(function (tradesResult) {
    console.log(tradesResult.records);
  })
  .catch(function (err) {
    console.log(err)
  })
```

### JavaScript Example Streaming Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var tradesHandler = function (tradeResponse) {
  console.log(tradeResponse);
};

var es = server.trades()
  .cursor('now')
  .stream({
  onmessage: tradesHandler
})
```

## Response

The list of trades. `base` and `counter` in the records will match the asset pair filter order. If an asset pair is not specified, the order is arbitrary.

### Example Response
```json
[
  {
    "_links": {
      "self": {
        "href": ""
      },
      "base": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M"
      },
      "counter": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GCRGG47RE7L2KVOEZCMYSQ6FTZF7KQJM3N3VA6IEF46GLPPAZHQGOSOA"
      },
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4173759023943681"
      }
    },
    "id": "4173759023943681-0",
    "paging_token": "4173759023943681-0",
    "ledger_close_time": "2021-06-17T07:29:30Z",
    "offer_id": "6",
    "base_offer_id": "6",
    "base_account": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
    "base_amount": "10.0000000",
    "base_asset_type": "credit_alphanum4",
    "base_asset_code": "UAH",
    "base_asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "counter_offer_id": "4615859777451331585",
    "counter_account": "GCRGG47RE7L2KVOEZCMYSQ6FTZF7KQJM3N3VA6IEF46GLPPAZHQGOSOA",
    "counter_amount": "5.0000000",
    "counter_asset_type": "native",
    "base_is_seller": true,
    "price": {
      "n": 1,
      "d": 2
    }
  },
  {
    "_links": {
      "self": {
        "href": ""
      },
      "base": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M"
      },
      "counter": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY"
      },
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4185647493419009"
      }
    },
    "id": "4185647493419009-0",
    "paging_token": "4185647493419009-0",
    "ledger_close_time": "2021-06-17T11:54:15Z",
    "offer_id": "7",
    "base_offer_id": "4615871665920806913",
    "base_account": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
    "base_amount": "1.0000000",
    "base_asset_type": "native",
    "counter_offer_id": "7",
    "counter_account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "counter_amount": "1.0000000",
    "counter_asset_type": "credit_alphanum4",
    "counter_asset_code": "USD",
    "counter_asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ",
    "base_is_seller": false,
    "price": {
      "n": 1,
      "d": 1
    }
  }
]

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
