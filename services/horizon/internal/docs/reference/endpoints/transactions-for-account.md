---
title: Transactions for Account
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=transactions&endpoint=for_account
---

This endpoint represents all [transactions](../resources/transaction.md) that affected a given [account](../resources/account.md).
This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to listen for new transactions as that affect a given account as they get made in the Stellar network.
If called in streaming mode Horizon will start at the earliest known transaction unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream transaction created since your request time.

## Request

```
GET /accounts/{account_id}/transactions{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account_id` | required, string | ID of an account | GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | 12884905984 |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/accounts/GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K/transactions?limit=1"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.transactions()
  .forAccount("GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K")
  .call()
  .then(function (accountResult) {
    console.log(accountResult);
  })
  .catch(function (err) {
    console.error(err);
  })
```

### JavaScript Streaming Example

```javascript
var StellarSdk = require('stellar-sdk')
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

var txHandler = function (txResponse) {
    console.log(txResponse);
};

var es = server.transactions()
    .forAccount("GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K")
    .stream({
        onmessage: txHandler
    })
```

## Response

This endpoint responds with a list of transactions that changed a given account's state. See [transaction resource](../resources/transaction.md) for reference.

### Example Response
```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCEZWKCA5VLDNRLN3RPRJMRZOX3Z6G5CHCGSNFHEYVXM3XOJMDS674JZ/transactions?cursor=&limit=1&order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCEZWKCA5VLDNRLN3RPRJMRZOX3Z6G5CHCGSNFHEYVXM3XOJMDS674JZ/transactions?cursor=25731149074432&limit=1&order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/accounts/GCEZWKCA5VLDNRLN3RPRJMRZOX3Z6G5CHCGSNFHEYVXM3XOJMDS674JZ/transactions?cursor=25731149074432&limit=1&order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://horizon-testnet.stellar.org/transactions/d486852ab6f96ec9f16d8972ef11199947a2c22132ac47f4bc645a186dc518c8"
          },
          "account": {
            "href": "https://horizon-testnet.stellar.org/accounts/GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K"
          },
          "ledger": {
            "href": "https://horizon-testnet.stellar.org/ledgers/5991"
          },
          "operations": {
            "href": "https://horizon-testnet.stellar.org/transactions/d486852ab6f96ec9f16d8972ef11199947a2c22132ac47f4bc645a186dc518c8/operations{?cursor,limit,order}",
            "templated": true
          },
          "effects": {
            "href": "https://horizon-testnet.stellar.org/transactions/d486852ab6f96ec9f16d8972ef11199947a2c22132ac47f4bc645a186dc518c8/effects{?cursor,limit,order}",
            "templated": true
          },
          "precedes": {
            "href": "https://horizon-testnet.stellar.org/transactions?order=asc&cursor=25731149074432"
          },
          "succeeds": {
            "href": "https://horizon-testnet.stellar.org/transactions?order=desc&cursor=25731149074432"
          }
        },
        "id": "d486852ab6f96ec9f16d8972ef11199947a2c22132ac47f4bc645a186dc518c8",
        "paging_token": "25731149074432",
        "hash": "d486852ab6f96ec9f16d8972ef11199947a2c22132ac47f4bc645a186dc518c8",
        "ledger": 5991,
        "created_at": "2017-03-20T23:39:43Z",
        "source_account": "GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K",
        "source_account_sequence": "10157597655056",
        "fee_paid": 100,
        "operation_count": 1,
        "envelope_xdr": "AAAAAGXNhLrhGtltTwCpmqlarh7s1DB2hIkbP//jgzn4Fos/AAAAZAAACT0AAAAQAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAiZsoQO1WNsVt3F8Usjl1958bojiNJpTkxW7N3clg5e8AAAAXSHboAAAAAAAAAAAB+BaLPwAAAEColV/6xTnefW6UqmMpu/Nn4flKsMxvlmQNSV6eBITUXonKWI5GhCC/HW1CRcnxmVR7NftBzkbBatnGgtWPraUH",
        "result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=",
        "result_meta_xdr": "AAAAAAAAAAEAAAACAAAAAAAAF2cAAAAAAAAAAImbKEDtVjbFbdxfFLI5dfefG6I4jSaU5MVuzd3JYOXvAAAAF0h26AAAABdnAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAF2cAAAAAAAAAAGXNhLrhGtltTwCpmqlarh7s1DB2hIkbP//jgzn4Fos/AAHFSsr0ucAAAAk9AAAAEAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA",
        "fee_meta_xdr": "AAAAAgAAAAMAABMBAAAAAAAAAABlzYS64RrZbU8AqZqpWq4e7NQwdoSJGz//44M5+BaLPwABxWITa6IkAAAJPQAAAA8AAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAABdnAAAAAAAAAABlzYS64RrZbU8AqZqpWq4e7NQwdoSJGz//44M5+BaLPwABxWITa6HAAAAJPQAAABAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
        "memo_type": "none",
        "signatures": [
          "qJVf+sU53n1ulKpjKbvzZ+H5SrDMb5ZkDUlengSE1F6JyliORoQgvx1tQkXJ8ZlUezX7Qc5GwWrZxoLVj62lBw=="
        ]
      }
    ]
  }
}
```

### Example Streaming Event

```json
{
  "_links": {
    "account": {
      "href": "/accounts/GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K"
    },
    "effects": {
      "href": "/transactions/fa78cb43d72171fdb2c6376be12d57daa787b1fa1a9fdd0e9453e1f41ee5f15a/effects{?cursor,limit,order}",
      "templated": true
    },
    "ledger": {
      "href": "/ledgers/146970"
    },
    "operations": {
      "href": "/transactions/fa78cb43d72171fdb2c6376be12d57daa787b1fa1a9fdd0e9453e1f41ee5f15a/operations{?cursor,limit,order}",
      "templated": true
    },
    "precedes": {
      "href": "/transactions?cursor=631231343497216\u0026order=asc"
    },
    "self": {
      "href": "/transactions/fa78cb43d72171fdb2c6376be12d57daa787b1fa1a9fdd0e9453e1f41ee5f15a"
    },
    "succeeds": {
      "href": "/transactions?cursor=631231343497216\u0026order=desc"
    }
  },
  "id": "fa78cb43d72171fdb2c6376be12d57daa787b1fa1a9fdd0e9453e1f41ee5f15a",
  "paging_token": "631231343497216",
  "hash": "fa78cb43d72171fdb2c6376be12d57daa787b1fa1a9fdd0e9453e1f41ee5f15a",
  "ledger": 146970,
  "created_at": "2015-09-24T10:07:09Z",
  "account": "GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K",
  "account_sequence": 279172874343,
  "max_fee": 0,
  "fee_paid": 0,
  "operation_count": 1,
  "envelope_xdr": "AAAAAGXNhLrhGtltTwCpmqlarh7s1DB2hIkbP//jgzn4Fos/AAAACgAAAEEAAABnAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAA2ddmTOFAgr21Crs2RXRGLhiAKxicZb/IERyEZL/Y2kUAAAAXSHboAAAAAAAAAAAB+BaLPwAAAECDEEZmzbgBr5fc3mfJsCjWPDtL6H8/vf16me121CC09ONyWJZnw0PUvp4qusmRwC6ZKfLDdk8F3Rq41s+yOgQD",
  "result_xdr": "AAAAAAAAAAoAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=",
  "result_meta_xdr": "AAAAAAAAAAEAAAACAAAAAAACPhoAAAAAAAAAANnXZkzhQIK9tQq7NkV0Ri4YgCsYnGW/yBEchGS/2NpFAAAAF0h26AAAAj4aAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQACPhoAAAAAAAAAAGXNhLrhGtltTwCpmqlarh7s1DB2hIkbP//jgzn4Fos/AABT8kS2c/oAAABBAAAAZwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA"
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no account whose ID matches the `account_id` argument.
