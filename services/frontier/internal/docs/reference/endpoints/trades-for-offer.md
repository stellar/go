---
title: Trades for Offer
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=trades&endpoint=for_offer
---

This endpoint represents all [trades](../resources/trade.md) for a given [offer](../resources/offer.md).

This endpoint can also be used in [streaming](../streaming.md) mode, making it possible to listen for new trades for the given offer as they occur on the DigitalBits network.
If called in streaming mode Frontier will start at the earliest known trade unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream trades created since your request time.
## Request

```
GET /offers/{offer_id}/trades{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `offer_id` | required, number | ID of an offer | 323223 |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. | 1623820974 |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/offers/6/trades"
```

### JavaScript Example Request

```js
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.trades()
    .forOffer(6)
    .call()
    .then(function (tradesResult) {
      console.log(JSON.stringify(tradesResult));
    })
    .catch(function (err) {
      console.error(err);
    })
```


## Response

This endpoint responds with a list of trades that consumed a given offer. See the [trade resource](../resources/trade.md) for reference.

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
    }
  ]
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no offer whose ID matches the `offer_id` argument.
