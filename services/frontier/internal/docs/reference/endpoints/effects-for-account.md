---
title: Effects for Account
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=effects&endpoint=for_account
---
This endpoint represents all [effects](../resources/effect.md) that changed a given
[account](../resources/account.md). It will return relevant effects from the creation of the
account to the current ledger.

This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to
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
| `account` | required, string | Account ID | `GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/effects?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.effects()
  .forAccount("GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY")
  .call()
  .then(function (effectResults) {
    // page 1
    console.log(JSON.strigify(effectResults.records))
  })
  .catch(function (err) {
    console.log(err)
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var effectHandler = function (effectResponse) {
  console.log(effectResponse);
};

var es = server.effects()
  .forAccount("GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY")
  .cursor('now')
  .stream({
    onmessage: effectHandler
  })
```

## Response

The list of effects.

### Example Response

```json
[
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4113023891410945"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4113023891410945-1"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4113023891410945-1"
      }
    },
    "id": "0004113023891410945-0000000001",
    "paging_token": "4113023891410945-1",
    "account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "type": "account_created",
    "type_i": 0,
    "created_at": "2021-06-16T08:55:24Z",
    "starting_balance": "10000.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4113023891410945"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4113023891410945-3"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4113023891410945-3"
      }
    },
    "id": "0004113023891410945-0000000003",
    "paging_token": "4113023891410945-3",
    "account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "type": "signer_created",
    "type_i": 10,
    "created_at": "2021-06-16T08:55:24Z",
    "weight": 1,
    "public_key": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "key": ""
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4113161330364417"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4113161330364417-1"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4113161330364417-1"
      }
    },
    "id": "0004113161330364417-0000000001",
    "paging_token": "4113161330364417-1",
    "account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "type": "trustline_created",
    "type_i": 20,
    "created_at": "2021-06-16T08:58:29Z",
    "asset_type": "credit_alphanum4",
    "asset_code": "USD",
    "asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ",
    "limit": "1000.0000000"
  }
]

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there are no effects for the given account.
