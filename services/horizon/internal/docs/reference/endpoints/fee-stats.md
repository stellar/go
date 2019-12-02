---
title: Fee Stats
clientData:
  laboratoryUrl:
---

This endpoint gives useful information about per-operation fee stats in the last 5 ledgers. It can be used to
predict a fee set on the transaction that will be submitted to the network.

Note: All `*_accepted_fee` fields are deprecated and  will be removed in Horizon `0.25.0`. Use the `max_fee` and `fee_charged` keys instead.

## Request

```
GET /fee_stats
```

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/fee_stats"
```

## Response

Response contains the following fields:

| Field | |
| - | - |
| last_ledger | Last ledger sequence number |
| last_ledger_base_fee | Base fee as defined in the last ledger |
| ledger_capacity_usage | Average capacity usage over the last 5 ledgers. (0 is no usage, 1.0 is completely full ledgers) |
| min_accepted_fee | Minimum accepted fee over the last 5 ledger. |
| mode_accepted_fee | Mode (most common value) of the maximum fee bid over the last 5 ledgers. |
| p10_accepted_fee | 10th percentile max fee over the last 5 ledger. |
| p20_accepted_fee | 20th percentile max fee over the last 5 ledger. |
| p30_accepted_fee | 30th percentile max fee over the last 5 ledger. |
| p40_accepted_fee | 40th percentile max fee over the last 5 ledger. |
| p50_accepted_fee | 50th percentile max fee over the last 5 ledger. |
| p60_accepted_fee | 60th percentile max fee over the last 5 ledger. |
| p70_accepted_fee | 70th percentile max fee over the last 5 ledger. |
| p80_accepted_fee | 80th percentile max fee over the last 5 ledger. |
| p90_accepted_fee | 90th percentile max fee over the last 5 ledger. |
| p95_accepted_fee | 95th percentile max fee over the last 5 ledger. |
| p99_accepted_fee | 99th percentile max fee over the last 5 ledger. |
| fee_charged      | fee charged object |
| max_fee          | max fee object |

### Fee Charged Object

Information about the fee charged for transactions in the last 5 ledgers.

| Field | |
| - | - |
| min | Minimum fee charged over the last 5 ledgers. |
| mode | Mode fee charged over the last 5 ledgers. |
| p10 | 10th percentile fee charged over the last 5 ledgers. |
| p20 | 20th percentile fee charged over the last 5 ledgers. |
| p30 | 30th percentile fee charged over the last 5 ledgers. |
| p40 | 40th percentile fee charged over the last 5 ledgers. |
| p50 | 50th percentile fee charged over the last 5 ledgers. |
| p60 | 60th percentile fee charged over the last 5 ledgers. |
| p70 | 70th percentile fee charged over the last 5 ledgers. |
| p80 | 80th percentile fee charged over the last 5 ledgers. |
| p90 | 90th percentile fee charged over the last 5 ledgers. |
| p95 | 95th percentile fee charged over the last 5 ledgers. |
| p99 | 99th percentile fee charged over the last 5 ledgers. |

Note: The difference between `fee_charged` and `max_fee` is that the former
represents the actual fee paid for the transaction while `max_fee` represents
the maximum bid the transaction creator was willing to pay for the transaction.

### Max Fee Object

Information about max fee bid for transactions over the last 5 ledgers.

| Field | |
| - | - |
| min | Minimum (lowest) value of the maximum fee bid over the last 5 ledgers. |
| mode | Mode max fee over the last 5 ledgers. |
| p10 | 10th percentile max fee over the last 5 ledgers. |
| p20 | 20th percentile max fee over the last 5 ledgers. |
| p30 | 30th percentile max fee over the last 5 ledgers. |
| p40 | 40th percentile max fee over the last 5 ledgers. |
| p50 | 50th percentile max fee over the last 5 ledgers. |
| p60 | 60th percentile max fee over the last 5 ledgers. |
| p70 | 70th percentile max fee over the last 5 ledgers. |
| p80 | 80th percentile max fee over the last 5 ledgers. |
| p90 | 90th percentile max fee over the last 5 ledgers. |
| p95 | 95th percentile max fee over the last 5 ledgers. |
| p99 | 99th percentile max fee over the last 5 ledgers. |


### Example Response

```json
{
  "last_ledger": "22606298",
  "last_ledger_base_fee": "100",
  "ledger_capacity_usage": "0.97",
  "min_accepted_fee": "100",
  "mode_accepted_fee": "250",
  "p10_accepted_fee": "100",
  "p20_accepted_fee": "100",
  "p30_accepted_fee": "250",
  "p40_accepted_fee": "250",
  "p50_accepted_fee": "250",
  "p60_accepted_fee": "1210",
  "p70_accepted_fee": "1221",
  "p80_accepted_fee": "1225",
  "p90_accepted_fee": "1225",
  "p95_accepted_fee": "1225",
  "p99_accepted_fee": "8000",
  "fee_charged": {
    "max": "100",
    "min": "100",
    "mode": "100",
    "p10": "100",
    "p20": "100",
    "p30": "100",
    "p40": "100",
    "p50": "100",
    "p60": "100",
    "p70": "100",
    "p80": "100",
    "p90": "100",
    "p95": "100",
    "p99": "100"
  },
  "max_fee": {
    "max": "100000",
    "min": "100",
    "mode": "100",
    "p10": "100",
    "p20": "100",
    "p30": "100",
    "p40": "100",
    "p50": "100",
    "p60": "100",
    "p70": "100",
    "p80": "100",
    "p90": "15000",
    "p95": "100000",
    "p99": "100000"
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard_Errors).
