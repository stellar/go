---
title: Transactions for Ledger
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=transactions&endpoint=for_ledger
---


This endpoint represents successful [transactions](../resources/transaction.md) in a given [ledger](../resources/ledger.md).

## Request

```
GET /ledgers/{id}/transactions{?cursor,limit,order,include_failed}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `id` | required, number | Ledger ID | `958064` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include failed transactions in results. | `true` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/ledgers/697121/transactions?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.transactions()
  .forLedger("958064")
  .limit("1")
  .call()
  .then(function (accountResults) {
    console.log(JSON.stringify(accountResults.records))
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a list of transactions in a given ledger. See [transaction
resource](../resources/transaction.md) for reference.

### Example Response

```json
[
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
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no ledgers whose sequence matches the `id` argument.
