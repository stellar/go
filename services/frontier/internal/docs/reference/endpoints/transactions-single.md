---
title: Transaction Details
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=transactions&endpoint=single
---

The transaction details endpoint provides information on a single
[transaction](../resources/transaction.md). The transaction hash provided in the `hash` argument
specifies which transaction to load.

### Warning - failed transactions

Transaction can be successful or failed (failed transactions are also included in DigitalBits ledger).
Always check it's status using `successful` field!

## Request

```
GET /transactions/{hash}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `hash` | required, string | A transaction hash, hex-encoded, lowercase. | `c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/transactions/c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.transactions()
  .transaction("c60f666cda020d13033bb44926adf7f6c2b659857f13959e3988351055c0b52f")
  .call()
  .then(function (transactionResult) {
    console.log(JSON.stringify(transactionResult))
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a single Transaction.  See [transaction resource](../resources/transaction.md) for reference.

### Example Response

```json
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

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no
  transaction whose ID matches the `hash` argument.
