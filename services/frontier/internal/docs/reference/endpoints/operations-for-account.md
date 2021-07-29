---
title: Operations for Account
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=operations&endpoint=for_account
---

This endpoint represents successful [operations](../resources/operation.md) that were included in valid [transactions](../resources/transaction.md) that affected a particular [account](../resources/account.md).

This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to listen for new operations that affect a given account as they happen.
If called in streaming mode Frontier will start at the earliest known operation unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream operations created since your request time.

## Request

```
GET /accounts/{account}/operations{?cursor,limit,order,include_failed}
```

### Arguments

| name     | notes                          | description                                                      | example                                                   |
| ------   | -------                        | -----------                                                      | -------                                                   |
| `account`| required, string               | Account ID                                                  | `GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY`|
| `?cursor`| optional, default _null_       | A paging token, specifying where to start returning records from.  When streaming this can be set to `now` to stream object created since your request time. | `1623820974`                                             |
| `?order` | optional, string, default `asc`| The order in which to return rows, "asc" or "desc".              | `asc`                                                     |
| `?limit` | optional, number, default `10` | Maximum number of records to return.                             | `200`
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include operations of failed transactions in results. | `true` |                                                     |
| `?join` | optional, string, default: _null_ | Set to `transactions` to include the transactions which created each of the operations in the response. | `transactions` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/operations"
```

### JavaScript Example Request

```js
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.operations()
  .forAccount("GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY")
  .call()
  .then(function (operationsResult) {
    console.log(JSON.stringify(operationsResult.records))
  })
  .catch(function (err) {
    console.log(err)
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var operationHandler = function (operationResponse) {
    console.log(operationResponse);
};

var es = server.operations()
    .forAccount("GAKLBGHNHFQ3BMUYG5KU4BEWO6EYQHZHAXEWC33W34PH2RBHZDSQBD75")
    .cursor('now')
    .stream({
        onmessage: operationHandler
    })
```

## Response

This endpoint responds with a list of operations that affected the given account. See [operation resource](../resources/operation.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4114853547479041"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4114853547479041/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4114853547479041"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4114853547479041"
      }
    },
    "id": "4114853547479041",
    "paging_token": "4114853547479041",
    "transaction_successful": true,
    "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "type": "payment",
    "type_i": 1,
    "created_at": "2021-06-16T09:36:04Z",
    "transaction_hash": "c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f",
    "asset_type": "credit_alphanum4",
    "asset_code": "HUF",
    "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "from": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "to": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "amount": "50000.0000000"
  },
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4115467727802369"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/896ebf830df1ad883603a9b08a3486110e8a81e877fb4078f8edb341d38266c9"
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/operations/4115467727802369/effects"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4115467727802369"
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4115467727802369"
      }
    },
    "id": "4115467727802369",
    "paging_token": "4115467727802369",
    "transaction_successful": true,
    "source_account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
    "type": "manage_data",
    "type_i": 10,
    "created_at": "2021-06-16T09:49:53Z",
    "transaction_hash": "896ebf830df1ad883603a9b08a3486110e8a81e877fb4078f8edb341d38266c9",
    "name": "user-id",
    "value": "WERCRm91bmRhdGlvbg=="
  }
]
```

### Example Streaming Event

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4114853547479041"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4114853547479041/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4114853547479041"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4114853547479041"
    }
  },
  "id": "4114853547479041",
  "paging_token": "4114853547479041",
  "transaction_successful": true,
  "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "type": "payment",
  "type_i": 1,
  "created_at": "2021-06-16T09:36:04Z",
  "transaction_hash": "c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f",
  "asset_type": "credit_alphanum4",
  "asset_code": "HUF",
  "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "from": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "to": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
  "amount": "50000.0000000"
}
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4115467727802369"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/896ebf830df1ad883603a9b08a3486110e8a81e877fb4078f8edb341d38266c9"
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/operations/4115467727802369/effects"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=4115467727802369"
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=4115467727802369"
    }
  },
  "id": "4115467727802369",
  "paging_token": "4115467727802369",
  "transaction_successful": true,
  "source_account": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
  "type": "manage_data",
  "type_i": 10,
  "created_at": "2021-06-16T09:49:53Z",
  "transaction_hash": "896ebf830df1ad883603a9b08a3486110e8a81e877fb4078f8edb341d38266c9",
  "name": "user-id",
  "value": "WERCRm91bmRhdGlvbg=="
}
```


## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no account whose ID matches the `account` argument.
