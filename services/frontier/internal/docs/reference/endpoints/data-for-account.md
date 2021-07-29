---
title: Data for Account
---

This endpoint represents a single [data](../resources/data.md) associated with a given [account](../resources/account.md).

## Request

```
GET /accounts/{account}/data/{key}
```

### Arguments

| name     | notes                          | description                                                      | example                                                   |
| ------   | -------                        | -----------                                                      | -------                                                   |
| `key`| required, string               | Key name | `user-id`|

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/data/user-id"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.accounts()
  .accountId("GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY")
  .call()
  .then(function (account) {
    return account.data({key: 'user-id'})
  })
  .then(function(dataValue) {
    console.log(JSON.stringify(dataValue))
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a value of the data field for the given account. See [data resource](../resources/data.md) for reference.

### Example Response

```json
{
  "value": "WERCRm91bmRhdGlvbg=="
}
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no account whose ID matches the `account` argument or there is no data field with a given key.
