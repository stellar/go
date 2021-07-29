---
title: All Effects
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=effects&endpoint=all
---

This endpoint represents all [effects](../resources/effect.md).

This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to listen for new effects as transactions happen in the DigitalBits network.
If called in streaming mode Frontier will start at the earliest known effect unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream effects created since your request time.

## Request

```
GET /effects{?cursor,limit,order}
```

## Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc".               | `asc`         |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/effects"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.effects()
  .call()
  .then(function (effectResults) {
    //page 1
    console.log(JSON.stringify(effectResults.records))
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
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631873-1"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631873-1"
      }
    },
    "id": "0000001099511631873-0000000001",
    "paging_token": "1099511631873-1",
    "account": "GDE3XSDA4G7MZJXZ6SYYD7CHQSOUFMEDTSU2WINVJ42DOFOCBTLGI5O4",
    "type": "account_created",
    "type_i": 0,
    "created_at": "2021-04-13T13:55:32Z",
    "starting_balance": "101.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631873-2"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631873-2"
      }
    },
    "id": "0000001099511631873-0000000002",
    "paging_token": "1099511631873-2",
    "account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "account_debited",
    "type_i": 3,
    "created_at": "2021-04-13T13:55:32Z",
    "asset_type": "native",
    "amount": "101.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631873"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631873-3"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631873-3"
      }
    },
    "id": "0000001099511631873-0000000003",
    "paging_token": "1099511631873-3",
    "account": "GDE3XSDA4G7MZJXZ6SYYD7CHQSOUFMEDTSU2WINVJ42DOFOCBTLGI5O4",
    "type": "signer_created",
    "type_i": 10,
    "created_at": "2021-04-13T13:55:32Z",
    "weight": 1,
    "public_key": "GDE3XSDA4G7MZJXZ6SYYD7CHQSOUFMEDTSU2WINVJ42DOFOCBTLGI5O4",
    "key": ""
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631874"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631874-1"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631874-1"
      }
    },
    "id": "0000001099511631874-0000000001",
    "paging_token": "1099511631874-1",
    "account": "GBPULLXKNDHPAP25N66JA4SH5SOQSNAIWKPVFTATMY6DDV43GBH2TUGV",
    "type": "account_created",
    "type_i": 0,
    "created_at": "2021-04-13T13:55:32Z",
    "starting_balance": "101.0000000"
  },
  {
    "_links": {
      "operation": {
        "href": "https://frontier.testnet.digitalbits.io/operations/1099511631874"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1099511631874-2"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1099511631874-2"
      }
    },
    "id": "0000001099511631874-0000000002",
    "paging_token": "1099511631874-2",
    "account": "GC3CLEUNQVWY36AHTGGX2NASAPHD6EBQXE63YH2B3PAASLCCIG4ELGTP",
    "type": "account_debited",
    "type_i": 3,
    "created_at": "2021-04-13T13:55:32Z",
    "asset_type": "native",
    "amount": "101.0000000"
  }
]
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there are no effects for the given account.
