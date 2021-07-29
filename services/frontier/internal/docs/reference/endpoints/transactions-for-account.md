---
title: Transactions for Account
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=transactions&endpoint=for_account
---

This endpoint represents successful [transactions](../resources/transaction.md) that affected a
given [account](../resources/account.md).  This endpoint can also be used in
[streaming](../streaming.md) mode so it is possible to use it to listen for new transactions that
affect a given account as they get made in the DigitalBits network.

If called in streaming mode Frontier will start at the earliest known transaction unless a `cursor`
is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to
only stream transaction created since your request time.

## Request

```
GET /accounts/{account_id}/transactions{?cursor,limit,order,include_failed}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account_id` | required, string | ID of an account | GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | 1623820974 |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include failed transactions in results. | `true` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK/transactions?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.transactions()
  .forAccount("GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK")
  .call()
  .then(function (accountResult) {
    console.log(JSON.stringify(accountResult));
  })
  .catch(function (err) {
    console.error(err);
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var txHandler = function (txResponse) {
  console.log(txResponse);
};

var es = server.transactions()
  .forAccount("GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK")
  .cursor('now')
  .stream({
    onmessage: txHandler
  })
```

## Response

This endpoint responds with a list of transactions that changed a given account's state. See
[transaction resource](../resources/transaction.md) for reference.

### Example Response
```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec"
      },
      "account": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
      },
      "ledger": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/956790"
      },
      "operations": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec/operations{?cursor,limit,order}",
        "templated": true
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec/effects{?cursor,limit,order}",
        "templated": true
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=asc&cursor=4109381759143936"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=desc&cursor=4109381759143936"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec"
      }
    },
    "id": "f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec",
    "paging_token": "4109381759143936",
    "successful": true,
    "hash": "f50342ab33a932dceb23bb44cbc03a13515563a07c51d3e76ecb430163345bec",
    "created_at": "2021-06-16T07:35:18Z",
    "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "source_account_sequence": "4109304449728513",
    "fee_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "fee_charged": "100",
    "max_fee": "100",
    "operation_count": 1,
    "envelope_xdr": "AAAAAgAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAAGQADplkAAAAAQAAAAEAAAAAAAAAAAAAAABgyaoVAAAAAAAAAAEAAAAAAAAAAQAAAACVjdVRSOUXRiw8RnB5Co3YaQY5JzT/cAe/GnqYO8a1ywAAAAFVQUgAAAAAAI8PAdmU/YeyLhQs8m5VLjvynfgW7JMaRRHdDpy5F08jAAAAAAX14QAAAAAAAAAAAbkXTyMAAABAW9j6rg13qMG1WWBuEH/0Yi14vPtUsxXiaip9W1YFafKGyQMk2Zmokz/QSukuaCRatuoNYcdrqrJWGNiOtM0rAQ==",
    "result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
    "result_meta_xdr": "AAAAAgAAAAIAAAADAA6ZdgAAAAAAAAAAjw8B2ZT9h7IuFCzyblUuO/Kd+BbskxpFEd0OnLkXTyMAAAAXSHbnnAAOmWQAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAA6ZdgAAAAAAAAAAjw8B2ZT9h7IuFCzyblUuO/Kd+BbskxpFEd0OnLkXTyMAAAAXSHbnnAAOmWQAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAgAAAAMADpl1AAAAAQAAAACVjdVRSOUXRiw8RnB5Co3YaQY5JzT/cAe/GnqYO8a1ywAAAAFVQUgAAAAAAI8PAdmU/YeyLhQs8m5VLjvynfgW7JMaRRHdDpy5F08jAAAAAAAAAAAAAAACVAvkAAAAAAEAAAAAAAAAAAAAAAEADpl2AAAAAQAAAACVjdVRSOUXRiw8RnB5Co3YaQY5JzT/cAe/GnqYO8a1ywAAAAFVQUgAAAAAAI8PAdmU/YeyLhQs8m5VLjvynfgW7JMaRRHdDpy5F08jAAAAAAX14QAAAAACVAvkAAAAAAEAAAAAAAAAAAAAAAA=",
    "fee_meta_xdr": "AAAABAAAAAMADplkAAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABdIdugAAA6ZZAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADpl2AAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABdIduecAA6ZZAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMADpl1AAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAx2oAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADpl2AAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAx4MAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
    "memo_type": "none",
    "signatures": [
      "W9j6rg13qMG1WWBuEH/0Yi14vPtUsxXiaip9W1YFafKGyQMk2Zmokz/QSukuaCRatuoNYcdrqrJWGNiOtM0rAQ=="
    ],
    "valid_after": "1970-01-01T00:00:00Z",
    "valid_before": "2021-06-16T07:36:53Z",
    "ledger_attr": 956790
  },
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f"
      },
      "account": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
      },
      "ledger": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/958064"
      },
      "operations": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f/operations{?cursor,limit,order}",
        "templated": true
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f/effects{?cursor,limit,order}",
        "templated": true
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=asc&cursor=4114853547479040"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=desc&cursor=4114853547479040"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f"
      }
    },
    "id": "c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f",
    "paging_token": "4114853547479040",
    "successful": true,
    "hash": "c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f",
    "created_at": "2021-06-16T09:36:04Z",
    "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "source_account_sequence": "4109304449728515",
    "fee_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "fee_charged": "100",
    "max_fee": "100",
    "operation_count": 1,
    "envelope_xdr": "AAAAAgAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAAGQADplkAAAAAwAAAAEAAAAAAAAAAAAAAABgycZkAAAAAAAAAAEAAAAAAAAAAQAAAADK462YFeuR6EO+Vst63pspzDtfuUukBAUFfeZKbxjkrQAAAAFIVUYAAAAAAI8PAdmU/YeyLhQs8m5VLjvynfgW7JMaRRHdDpy5F08jAAAAdGpSiAAAAAAAAAAAAbkXTyMAAABA7wnV5V0MZeIdMojlytJWWD6kDFTlA70EaQCPQ/N+4+PKtAGCYn0KVXvSKxL59eSFwZCviQ2dgep97MfAgN32Aw==",
    "result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
    "result_meta_xdr": "AAAAAgAAAAIAAAADAA6ecAAAAAAAAAAAjw8B2ZT9h7IuFCzyblUuO/Kd+BbskxpFEd0OnLkXTyMAAAAXSHbm1AAOmWQAAAACAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAA6ecAAAAAAAAAAAjw8B2ZT9h7IuFCzyblUuO/Kd+BbskxpFEd0OnLkXTyMAAAAXSHbm1AAOmWQAAAADAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAgAAAAMADp5vAAAAAQAAAADK462YFeuR6EO+Vst63pspzDtfuUukBAUFfeZKbxjkrQAAAAFIVUYAAAAAAI8PAdmU/YeyLhQs8m5VLjvynfgW7JMaRRHdDpy5F08jAAAAAAAAAAB//////////wAAAAEAAAAAAAAAAAAAAAEADp5wAAAAAQAAAADK462YFeuR6EO+Vst63pspzDtfuUukBAUFfeZKbxjkrQAAAAFIVUYAAAAAAI8PAdmU/YeyLhQs8m5VLjvynfgW7JMaRRHdDpy5F08jAAAAdGpSiAB//////////wAAAAEAAAAAAAAAAAAAAAA=",
    "fee_meta_xdr": "AAAABAAAAAMADp5kAAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABdIduc4AA6ZZAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADp5wAAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABdIdubUAA6ZZAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMADp5vAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAyakAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADp5wAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAycIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
    "memo_type": "none",
    "signatures": [
      "7wnV5V0MZeIdMojlytJWWD6kDFTlA70EaQCPQ/N+4+PKtAGCYn0KVXvSKxL59eSFwZCviQ2dgep97MfAgN32Aw=="
    ],
    "valid_after": "1970-01-01T00:00:00Z",
    "valid_before": "2021-06-16T09:37:40Z",
    "ledger_attr": 958064
  }
]

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no account whose ID matches the `account_id` argument.
