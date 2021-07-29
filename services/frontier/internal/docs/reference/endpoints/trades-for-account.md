---
title: Trades for Account
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=trades&endpoint=for_account
---

This endpoint represents all [trades](../resources/trade.md) that affect a given [account](../resources/account.md).

This endpoint can also be used in [streaming](../streaming.md) mode, making it possible to listen for new trades that affect the given account as they occur on the DigitalBits network.
If called in streaming mode Frontier will start at the earliest known trade unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream trades created since your request time.

## Request

```
GET /accounts/{account_id}/trades{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account_id` | required, string | ID of an account | `GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY` |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | 1623820974 |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/trades?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.trades()
  .forAccount("GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY")
  .call()
  .then(function (accountResult) {
    console.log(JSON.stringify(accountResult));
  })
  .catch(function (err) {
    console.error(err);
  })
```


## Response

This endpoint responds with a list of trades that changed a given account's state. See the [trade resource](../resources/trade.md) for reference.

### Example Response

```json
{
  "records": [
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
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no account whose ID matches the `account_id` argument.
