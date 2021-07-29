---
title: Ledger Details
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=ledgers&endpoint=single
---

The ledger details endpoint provides information on a single [ledger](../resources/ledger.md).

## Request

```
GET /ledgers/{sequence}
```

### Arguments

|  name  |  notes  | description | example |
| ------ | ------- | ----------- | ------- |
| `sequence` | required, number | Ledger Sequence | `957773` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/ledgers/957773"
```

### JavaScript Example Request

```js
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.ledgers()
  .ledger('957773')
  .call()
  .then(function(ledgerResult) {
    console.log(JSON.stringify(ledgerResult))
  })
  .catch(function(err) {
    console.log(err)
  })

```
## Response

This endpoint responds with a single Ledger.  See [ledger resource](../resources/ledger.md) for reference.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/957773"
    },
    "transactions": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/957773/transactions{?cursor,limit,order}",
      "templated": true
    },
    "operations": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/957773/operations{?cursor,limit,order}",
      "templated": true
    },
    "payments": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/957773/payments{?cursor,limit,order}",
      "templated": true
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/957773/effects{?cursor,limit,order}",
      "templated": true
    }
  },
  "id": "65e18185725ec118092b4915e341c7c75917738b238bee8998deead91946be75",
  "paging_token": "4113603711991808",
  "hash": "65e18185725ec118092b4915e341c7c75917738b238bee8998deead91946be75",
  "prev_hash": "d5c1154590395e887757be1809a48135d2ca8822e505ef50f52065f0cb21c1eb",
  "sequence": 957773,
  "successful_transaction_count": 1,
  "failed_transaction_count": 0,
  "operation_count": 1,
  "tx_set_operation_count": 1,
  "closed_at": "2021-06-16T09:08:23Z",
  "total_coins": "20000000000.0000000",
  "fee_pool": "0.0206000",
  "base_fee_in_nibbs": 100,
  "base_reserve_in_nibbs": 100000000,
  "max_tx_set_size": 100,
  "protocol_version": 15,
  "header_xdr": "AAAAD9XBFUWQOV6Id1e+GAmkgTXSyogi5QXvUPUgZfDLIcHrux5WF1JwRkTjTUEZ93mXRq4N8U6Xp53revUuRqqGiksAAAAAYMm/hwAAAAAAAAABAAAAAKvZnpGcHWYsIfQkvokpnA88t6aedQMkQ3LW/icyV30jAAAAQNqKT73RmwY7exn3h85m8RAlZ57SXrMH/TfYk6Gxvy1owUiHbyL1m1LcWDfjGLY429i3Ppwqb+XW35132vSOFgjTwPMb1bVYrBmvDFs/huBlb3dyFHWxx0guIrpv+rJCRlm1t9zYqfOiTdbG7MebuJxNp/r7H+dD0foh9v12OJ3FAA6dTQLGivC7FAAAAAAAAAADJLAAAAAAAAAAAAAAAAAAAABkBfXhAAAAAGSaOd07uLsDMIPf6nvuLuj6ev7+suXe3mfFES+inzTMUbGnoJtIrqPU36tDFU4XORBgsCvIi04GG/A0tVIWclCY2pL7Nkua71s2zrhLvP2xk17wI1QdTs2NbP8p4hUvqO96TzEzCTu1IfbrP9QD0x0cN77mrkt2Hhi4BP6sYQcbDgAAAAA="
}

```

## Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no ledger whose sequence number matches the `sequence` argument.
