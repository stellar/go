This endpoint represents all [effects](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/resources/effect) that changed a given
[account](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/resources/account). It will return relevant effects from the creation of the
account to the current ledger.

This endpoint can also be used in [streaming](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/streaming) mode so it is possible to use it to
listen for new effects as transactions happen in the DigitalBits network.
If called in streaming mode Frontier will start at the earliest known effect unless a `cursor` is
set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only
stream effects created since your request time.

## Request

```
GET /accounts/{account}/effects{?cursor,limit,order}
```

## Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account` | required, string | Account ID | `GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | `12884905984` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36/effects?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.effects()
  .forAccount("GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36")
  .call()
  .then(function (effectResults) {
    // page 1
    console.log(effectResults.records)
  })
  .catch(function (err) {
    console.log(err)
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var effectHandler = function (effectResponse) {
  console.log(effectResponse);
};

var es = server.effects()
  .forAccount("GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36")
  .cursor('now')
  .stream({
    onmessage: effectHandler
  })
```

## Response

The list of effects.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36/effects?cursor=&limit=1&order=asc"
    },
    "next": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36/effects?cursor=1919197546291201-1&limit=1&order=asc"
    },
    "prev": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36/effects?cursor=1919197546291201-1&limit=1&order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "operation": {
            "href": "https://frontier.testnet.digitalbits.io/operations/1919197546291201"
          },
          "succeeds": {
            "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1919197546291201-1"
          },
          "precedes": {
            "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1919197546291201-1"
          }
        },
        "id": "0001919197546291201-0000000001",
        "paging_token": "1919197546291201-1",
        "account": "GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36",
        "type": "account_created",
        "type_i": 0,
        "created_at": "2019-03-25T22:43:38Z",
        "starting_balance": "10000.0000000"
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/errors#standard-errors).
- [not_found](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/errors/not-found): A `not_found` error will be returned if there are no effects for the given account.
