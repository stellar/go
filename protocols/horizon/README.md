# Horizon Protocol Changelog

Any changes to the Horizon Public API should be included in this doc.

# SDK support

We started tracking SDK support at version 0.12.3. Support for 0.12.3 means that SDK can correctly:

* Send requests using all available query params / POST params / headers,
* Parse all fields in responses structs and headers.

For each new version we will only track changes from the previous version.

| Resource                                      | Changes                                      | Go SDK <sup>1</sup>            | JS SDK             | Java SDK                                          |
|:----------------------------------------------|:---------------------------------------------|:-------------------------------|:-------------------|:--------------------------------------------------|
| **0.12.3**                                    |                                              |                                |                    |                                                   |
| `GET /`                                       |                                              | +<br />(some `_links` missing) | x                  | x                                                 |
| `GET /metrics`                                |                                              | x                              | x                  | x                                                 |
| `GET /ledgers`                                |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers` SSE                            |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers/{ledger_id}`                    |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers/{ledger_id}/transactions`       |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers/{ledger_id}/transactions` SSE   |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers/{ledger_id}/operations`         |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers/{ledger_id}/operations` SSE     |                                              | x                              | 0.8.2              | x                                                 |
| `GET /ledgers/{ledger_id}/payments`           |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers/{ledger_id}/payments` SSE       |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /ledgers/{ledger_id}/effects`            |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /ledgers/{ledger_id}/effects` SSE        |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /accounts/{account_id}`                  |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /accounts/{account_id}/transactions`     |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /accounts/{account_id}/transactions` SSE |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /accounts/{account_id}/operations`       |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /accounts/{account_id}/operations` SSE   |                                              | x                              | 0.8.2              | x                                                 |
| `GET /accounts/{account_id}/payments`         |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /accounts/{account_id}/payments` SSE     |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /accounts/{account_id}/effects`          |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /accounts/{account_id}/effects` SSE      |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /accounts/{account_id}/offers`           |                                              | +                              | x                  | 0.2.0                                             |
| `GET /accounts/{account_id}/trades`           |                                              | x                              | 0.8.2              | x                                                 |
| `GET /accounts/{account_id}/data/{key}`       |                                              | x                              | x                  | x                                                 |
| `POST /transactions`                          |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /transactions`                           |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /transactions` SSE                       |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /transactions/{tx_id}`                   |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /transactions/{tx_id}/operations`        |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /transactions/{tx_id}/operations` SSE    |                                              | x                              | 0.8.2              | x                                                 |
| `GET /transactions/{tx_id}/payments`          |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /transactions/{tx_id}/effects`           |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /transactions/{tx_id}/effects` SSE       |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /operations`                             |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /operations` SSE                         |                                              | x                              | 0.8.2              | x                                                 |
| `GET /operations/{op_id}`                     |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /operations/{op_id}/effects`             |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /operations/{op_id}/effects` SSE         |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /payments`                               |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /payments` SSE                           |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /effects`                                |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /effects` SSE                            |                                              | x                              | 0.8.2              | 0.2.0<br />(no support for data, inflation types) |
| `GET /trades`                                 |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /trades_aggregations`                    |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /offers`                                 |                                              | x                              | x                  | 0.2.0                                             |
| `GET /offers` SSE                             |                                              | x                              | x                  | x                                                 |
| `GET /offers/{offer_id}`                      |                                              | x                              | x                  | x                                                 |
| `GET /offers/{offer_id}/trades`               |                                              | x                              | 0.8.2              | x                                                 |
| `GET /order_book`                             |                                              | +                              | 0.8.2              | 0.2.0                                             |
| `GET /order_book` SSE                         |                                              | x                              | 0.8.2              | x                                                 |
| `GET /paths`                                  |                                              | x                              | 0.8.2              | 0.2.0                                             |
| `GET /assets`                                 |                                              | x                              | 0.8.2              | 0.2.0                                             |
| [**0.13.0**](#0130) (changes only)            |                                              |                                |                    |                                                   |
| `GET /assets`                                 | `amount` can be larger than `MAX_INT64`/10^7 | +                              | 0.8.2 <sup>2</sup> | 0.2.0                                             |
| `GET /ledgers/{ledger_id}/effects`            | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /ledgers/{ledger_id}/effects` SSE        | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /accounts/{account_id}/effects`          | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /accounts/{account_id}/effects` SSE      | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /transactions/{tx_id}/effects`           | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /transactions/{tx_id}/effects` SSE       | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /operations/{op_id}/effects`             | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /operations/{op_id}/effects` SSE         | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /effects`                                | `created_at` field added                     | +                              | 0.8.2 <sup>2</sup> | x                                                 |
| `GET /effects` SSE                            | `created_at` field ad                        |                                |                    |                                                   |ded | +                              | 0.8.2 <sup>2</sup> | x                                                 |

<sup>1</sup> We don't do proper versioning for GO SDK yet. `+` means implemented in `master` branch.

<sup>2</sup> Native JSON support in JS, no changes needed.

# Changes

## 0.13.0

- `amount` field in `/assets` is now a String (to support Stellar amounts larger than `int64`).
- Effect resource contains a new `created_at` field.
