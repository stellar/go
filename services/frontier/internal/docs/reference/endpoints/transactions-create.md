---
title: Post Transaction
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=transactions&endpoint=create
---

Posts a new [transaction](../resources/transaction.md) to the DigitalBits Network.
Note that creating a valid transaction and signing it properly is the
responsibility of your client library.

Transaction submission and the subsequent validation and inclusion into the
DigitalBits Network's ledger is a [complicated and asynchronous
process](https://developers.digitalbits.io/guides/concepts/transactions.html#life-cycle).
To reduce the complexity, frontier manages these asynchronous processes for the
client and will wait to hear results from the DigitalBits Network before returning
an HTTP response to a client.

Transaction submission to frontier aims to be
[idempotent](https://en.wikipedia.org/wiki/Idempotence#Computer_science_meaning):
a client can submit a given transaction to frontier more than once and frontier
will behave the same each time.  If the transaction has already been
successfully applied to the ledger, frontier will simply return the saved result
and not attempt to submit the transaction again. Only in cases where a
transaction's status is unknown (and thus will have a chance of being included
into a ledger) will a resubmission to the network occur.

Information about [building transactions](https://developers.digitalbits.io/xdb-digitalbits-base/reference/building-transactions.html) in JavaScript.

### Timeout

If you are encountering this error it means that either:

* Frontier has not received a confirmation from the Core server that the transaction you are trying to submit to the network was included in a ledger in a timely manner or:
* Frontier has not sent a response to a reverse-proxy before in a specified time.

The former case may happen because there was no room for your transaction in the 3 consecutive ledgers. In such case, Core server removes a transaction from a queue. To solve this you can either:

* Keep resubmitting the same transaction (with the same sequence number) and wait until it finally is added to a new ledger or:
* Increase the [fee](https://developers.digitalbits.io/guides/concepts/fees.html).

## Request

```
POST /transactions
```

### Arguments

| name | loc  |  notes   |         example        | description |
| ---- | ---- | -------- | ---------------------- | ----------- |
| `tx` | body | required | `AAAAAO`....`f4yDBA==` | Base64 representation of transaction envelope [XDR](../xdr.md) |


### curl Example Request

```sh
curl -X POST \
     -F "tx=AAAAAgAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAAGQADplkAAAABAAAAAEAAAAAAAAAAAAAAABgye7BAAAAAAAAAAEAAAAAAAAAAQAAAAB5HNJF2fJhAUHdQs1L8HHOLqp12tebmDiPcu4HXhKOXAAAAAAAAAAAC+vCAAAAAAAAAAABuRdPIwAAAEA3j3oirwDyN1yR0gz3fPkgMonyVfchwftbo+EOvgrLYvADNT0Uqsa0LmM7/LQBeitk2v5GEiMT7PCzlORbnFsP" \
  "https://frontier.testnet.digitalbits.io/transactions"
```

## Response

A successful response (i.e. any response with a successful HTTP response code)
indicates that the transaction was successful and has been included into the
ledger.

If the transaction failed or errored, then an error response will be returned. Please see the errors section below.

### Attributes

The response will include all fields from the [transaction resource](../resources/transaction.md).

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/f5d126b3b2e870d3ec5f2ed7d11135cf4cefc95459d5ffb8c67dcd3731c08cc6"
    },
    "account": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
    },
    "ledger": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/959837"
    },
    "operations": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/f5d126b3b2e870d3ec5f2ed7d11135cf4cefc95459d5ffb8c67dcd3731c08cc6/operations{?cursor,limit,order}",
      "templated": true
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/f5d126b3b2e870d3ec5f2ed7d11135cf4cefc95459d5ffb8c67dcd3731c08cc6/effects{?cursor,limit,order}",
      "templated": true
    },
    "precedes": {
      "href": "https://frontier.testnet.digitalbits.io/transactions?order=asc\u0026cursor=4122468524494848"
    },
    "succeeds": {
      "href": "https://frontier.testnet.digitalbits.io/transactions?order=desc\u0026cursor=4122468524494848"
    },
    "transaction": {
      "href": "https://frontier.testnet.digitalbits.io/transactions/f5d126b3b2e870d3ec5f2ed7d11135cf4cefc95459d5ffb8c67dcd3731c08cc6"
    }
  },
  "id": "f5d126b3b2e870d3ec5f2ed7d11135cf4cefc95459d5ffb8c67dcd3731c08cc6",
  "paging_token": "4122468524494848",
  "successful": true,
  "hash": "f5d126b3b2e870d3ec5f2ed7d11135cf4cefc95459d5ffb8c67dcd3731c08cc6",
  "ledger": 959837,
  "created_at": "2021-06-16T12:26:02Z",
  "source_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "source_account_sequence": "4109304449728516",
  "fee_account": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
  "fee_charged": "100",
  "max_fee": "100",
  "operation_count": 1,
  "envelope_xdr": "AAAAAgAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAAGQADplkAAAABAAAAAEAAAAAAAAAAAAAAABgye7BAAAAAAAAAAEAAAAAAAAAAQAAAAB5HNJF2fJhAUHdQs1L8HHOLqp12tebmDiPcu4HXhKOXAAAAAAAAAAAC+vCAAAAAAAAAAABuRdPIwAAAEA3j3oirwDyN1yR0gz3fPkgMonyVfchwftbo+EOvgrLYvADNT0Uqsa0LmM7/LQBeitk2v5GEiMT7PCzlORbnFsP",
  "result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
  "result_meta_xdr": "AAAAAgAAAAIAAAADAA6lXQAAAAAAAAAAjw8B2ZT9h7IuFCzyblUuO/Kd+BbskxpFEd0OnLkXTyMAAAAXSHbmcAAOmWQAAAADAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAA6lXQAAAAAAAAAAjw8B2ZT9h7IuFCzyblUuO/Kd+BbskxpFEd0OnLkXTyMAAAAXSHbmcAAOmWQAAAAEAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAABAAAAAMADpznAAAAAAAAAAB5HNJF2fJhAUHdQs1L8HHOLqp12tebmDiPcu4HXhKOXAAAABdIduecAA6cxAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADqVdAAAAAAAAAAB5HNJF2fJhAUHdQs1L8HHOLqp12tebmDiPcu4HXhKOXAAAABdUYqmcAA6cxAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMADqVdAAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABdIduZwAA6ZZAAAAAQAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADqVdAAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABc8iyRwAA6ZZAAAAAQAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAA=",
  "fee_meta_xdr": "AAAABAAAAAMADp5wAAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABdIdubUAA6ZZAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADqVdAAAAAAAAAACPDwHZlP2Hsi4ULPJuVS478p34FuyTGkUR3Q6cuRdPIwAAABdIduZwAA6ZZAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMADp89AAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAynEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADqVdAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAyooAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
  "memo_type": "none",
  "signatures": [
    "N496Iq8A8jdckdIM93z5IDKJ8lX3IcH7W6PhDr4Ky2LwAzU9FKrGtC5jO/y0AXorZNr+RhIjE+zws5TkW5xbDw=="
  ],
  "valid_after": "1970-01-01T00:00:00Z",
  "valid_before": "2021-06-16T12:29:53Z"
}
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [transaction_failed](../errors/transaction-failed.md): The transaction failed and could not be applied to the ledger.
- [transaction_malformed](../errors/transaction-malformed.md): The transaction could not be decoded and was not submitted to the network.
- [timeout](../errors/timeout.md): No response from the Core server in a timely manner. Please check "Timeout" section above.
