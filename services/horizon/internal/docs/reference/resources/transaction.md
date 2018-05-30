---
title: Transaction
---

**Transactions** are the basic unit of change in the Stellar Network.

A transaction is a grouping of [operations](./operation.md).

To learn more about the concept of transactions in the Stellar network, take a look at the [Stellar transactions concept guide](https://www.stellar.org/developers/learn/concepts/transactions.html).

## Attributes

|    Attribute     |  Type  |                                                                                                                                |
| ---------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------ |
| id               | string | The canonical id of this transaction, suitable for use as the :id parameter for url templates that require a transaction's ID. |
| paging_token     | string | A [paging token](./page.md) suitable for use as the `cursor` parameter to transaction collection resources.                   |
| hash             | string | A hex-encoded SHA-256 hash of the transaction's [XDR](../../learn/xdr.md)-encoded form.                                                              |
| ledger           | number | Sequence number of the ledger in which this transaction was applied.       |
| account          | string |                                                                                                                                |
| account_sequence | number |                                                                                                                                |
| fee_paid         | number | The fee paid by the source account of this transaction when the transaction was applied to the ledger.                         |
| operation_count  | number | The number of operations that are contained within this transaction.                                                           |
| envelope_xdr     | string | A base64 encoded string of the raw `TransactionEnvelope` xdr struct for this transaction                                       |
| result_xdr       | string | A base64 encoded string of the raw `TransactionResultPair` xdr struct for this transaction                                     |
| result_meta_xdr  | string | A base64 encoded string of the raw `TransactionMeta` xdr struct for this transaction                                           |
| fee_meta_xdr  | string | A base64 encoded string of the raw `LedgerEntryChanges` xdr struct produced by taking fees for this transaction.                                           |

## Links

|                   rel                    |                                           Example                                           |                             Description                          |
| ---------------------------------------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| self | `/transactions/6391dd190f15f7d1665ba53c63842e368f485651a53d8d852ed442a446d1c69a`|  |
| account | `/accounts/GCEZWKCA5VLDNRLN3RPRJMRZOX3Z6G5CHCGSNFHEYVXM3XOJMDS674JZ` | The source [account](../accounts-single.md) for this transaction. |
| ledger | `/ledgers/3` | The [ledger](../ledgers-single.md) in which this transaction was applied. |
| operations | `/transactions/6391dd190f15f7d1665ba53c63842e368f485651a53d8d852ed442a446d1c69a/operations` | [Operations](../operations-for-transaction.md) included in this transaction. |
| effects | `/transactions/6391dd190f15f7d1665ba53c63842e368f485651a53d8d852ed442a446d1c69a/effects` | [Effects](../effects-for-transaction.md) which resulted by operations in this transaction. |
| precedes | `/transactions?cursor=12884905984&order=asc` | A collection of transactions that occur after this transaction. |
| succeeds | `/transactions?cursor=12884905984&order=desc` | A collection of transactions that occur before this transaction. |

## Example

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
  "fee_paid": 0,
  "operation_count": 1,
  "envelope_xdr": "AAAAAGXNhLrhGtltTwCpmqlarh7s1DB2hIkbP//jgzn4Fos/AAAACgAAAEEAAABnAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAA2ddmTOFAgr21Crs2RXRGLhiAKxicZb/IERyEZL/Y2kUAAAAXSHboAAAAAAAAAAAB+BaLPwAAAECDEEZmzbgBr5fc3mfJsCjWPDtL6H8/vf16me121CC09ONyWJZnw0PUvp4qusmRwC6ZKfLDdk8F3Rq41s+yOgQD",
  "result_xdr": "AAAAAAAAAAoAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=",
  "result_meta_xdr": "AAAAAAAAAAEAAAACAAAAAAACPhoAAAAAAAAAANnXZkzhQIK9tQq7NkV0Ri4YgCsYnGW/yBEchGS/2NpFAAAAF0h26AAAAj4aAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQACPhoAAAAAAAAAAGXNhLrhGtltTwCpmqlarh7s1DB2hIkbP//jgzn4Fos/AABT8kS2c/oAAABBAAAAZwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA"
}
```

## Endpoints

|  Resource                |    Type    |    Resource URI Template             |
| ------------------------ | ---------- | ------------------------------------ |
| [All Transactions](../transactions-all.md)     | Collection | `/transactions` (`GET`) |
| [Post Transaction](../transactions-create.md)     | Action | `/transactions`  (`POST`) |
| [Transaction Details](../transactions-single.md)  | Single     | `/transactions/:id` |
| [Account Transactions](../transactions-for-account.md) | Collection | `/accounts/:account_id/transactions` |
| [Ledger Transactions](../transactions-for-ledger.md)  | Collection | `/ledgers/:ledger_id/transactions`   |


## Submitting transactions
To submit a new transaction to Stellar network, it must first be built and signed locally. Then you can submit a hex representation of your transaction’s [XDR](../../learn/xdr.md) to the `/transactions` endpoint. Read more about submitting transactions in [Post Transaction](../transactions-create.md) doc.
