---
title: All Transactions
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=transactions&endpoint=all
---


This endpoint represents all successful [transactions](../resources/transaction.md).
Please note that this endpoint returns failed transactions that are included in the ledger if
`include_failed` parameter is `true` and Frontier is ingesting failed transactions.
This endpoint can also be used in [streaming](../streaming.md) mode. This makes it possible to use
it to listen for new transactions as they get made in the DigitalBits network. If called in streaming
mode Frontier will start at the earliest known transaction unless a `cursor` is set. In that case it
will start from the `cursor`. You can also set `cursor` value to `now` to only stream transaction
created since your request time.

## Request

```
GET /transactions{?cursor,limit,order,include_failed}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include failed transactions in results. | `true` |

### curl Example Request
# Retrieve the 200 latest transactions, ordered chronologically:

```sh
curl "https://frontier.testnet.digitalbits.io/transactions?limit=200&order=desc"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.transactions()
  .call()
  .then(function (transactionResult) {
    //page 1
    console.log(JSON.stringify(transactionResult.records));
    return transactionResult.next();
  })
  .then(function (transactionResult) {
    console.log(JSON.stringify(transactionResult.records));
  })
  .catch(function (err) {
    console.log(err)
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
  .cursor('now')
  .stream({
      onmessage: txHandler
  })
```

## Response

If called normally this endpoint responds with a [page](../resources/page.md) of transactions.
If called in streaming mode the transaction resources are returned individually.
See [transaction resource](../resources/transaction.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2"
      },
      "account": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GCAL6H3K4I6YZVGFRXILANRQA6ZUJH742ABERS5RA474DIACIN6T43OM"
      },
      "ledger": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/933127"
      },
      "operations": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2/operations{?cursor,limit,order}",
        "templated": true
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2/effects{?cursor,limit,order}",
        "templated": true
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=asc&cursor=4007749948018688"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=desc&cursor=4007749948018688"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2"
      }
    },
    "id": "26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2",
    "paging_token": "4007749948018688",
    "successful": true,
    "hash": "26070e6e7f9c34fd69915de6b2e01b9c4db4053a747019686d2a76437c8919d2",
    "created_at": "2021-06-14T17:53:02Z",
    "source_account": "GCAL6H3K4I6YZVGFRXILANRQA6ZUJH742ABERS5RA474DIACIN6T43OM",
    "source_account_sequence": "1099511627777",
    "fee_account": "GCAL6H3K4I6YZVGFRXILANRQA6ZUJH742ABERS5RA474DIACIN6T43OM",
    "fee_charged": "300",
    "max_fee": "300",
    "operation_count": 1,
    "envelope_xdr": "AAAAAgAAAACAvx9q4j2M1MWN0LA2MAezRJ/80AJIy7EHP8GgAkN9PgAAASwAAAEAAAAAAQAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAABAAAAALYlko2FbY34B5mNfTQSA84/EDC5PbwfQdvACSxCQbhFAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABdIdugAAAAAAAAAAAICQ30+AAAAQCmdeGqUTvsUeRFjQABMVyRFL2IyUKzIDrsRXyXA29sYmTQwSwvlof7MmghCqyUqmzHfzHdgiPSyuK+17T/LswpCQbhFAAAAQBEbA/9+Nh/sR6YixIAQM2sBibkvFOu4U9W5h13dWi/NFMPihDshyv4MNBgfXVI0A3pglNiShaBkgxWikVPGigU=",
    "result_xdr": "AAAAAAAAASwAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=",
    "result_meta_xdr": "AAAAAgAAAAIAAAADAA49BwAAAAAAAAAAgL8fauI9jNTFjdCwNjAHs0Sf/NACSMuxBz/BoAJDfT4AAAAAPDNfVAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAA49BwAAAAAAAAAAgL8fauI9jNTFjdCwNjAHs0Sf/NACSMuxBz/BoAJDfT4AAAAAPDNfVAAAAQAAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMADYQDAAAAAAAAAAC2JZKNhW2N+AeZjX00EgPOPxAwuT28H0HbwAksQkG4RQLGiHdubrLAAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj0HAAAAAAAAAAC2JZKNhW2N+AeZjX00EgPOPxAwuT28H0HbwAksQkG4RQLGiGAl98rAAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAADj0HAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABdIdugAAA49BwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAA=",
    "fee_meta_xdr": "AAAABAAAAAMAAAEAAAAAAAAAAACAvx9q4j2M1MWN0LA2MAezRJ/80AJIy7EHP8GgAkN9PgAAAAA8M2CAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj0HAAAAAAAAAACAvx9q4j2M1MWN0LA2MAezRJ/80AJIy7EHP8GgAkN9PgAAAAA8M19UAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMADYQDAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAxY8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj0HAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAxdoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
    "memo_type": "none",
    "signatures": [
      "KZ14apRO+xR5EWNAAExXJEUvYjJQrMgOuxFfJcDb2xiZNDBLC+Wh/syaCEKrJSqbMd/Md2CI9LK4r7XtP8uzCg==",
      "ERsD/342H+xHpiLEgBAzawGJuS8U67hT1bmHXd1aL80Uw+KEOyHK/gw0GB9dUjQDemCU2JKFoGSDFaKRU8aKBQ=="
    ],
    "valid_after": "1970-01-01T00:00:00Z",
    "ledger_attr": 933127
  },
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/098824f8a801f5da220363a9039c2f253108d00305217c6360f125699324f2a8"
      },
      "account": {
        "href": "https://frontier.testnet.digitalbits.io/accounts/GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT"
      },
      "ledger": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/933683"
      },
      "operations": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/098824f8a801f5da220363a9039c2f253108d00305217c6360f125699324f2a8/operations{?cursor,limit,order}",
        "templated": true
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/098824f8a801f5da220363a9039c2f253108d00305217c6360f125699324f2a8/effects{?cursor,limit,order}",
        "templated": true
      },
      "precedes": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=asc&cursor=4010137949835264"
      },
      "succeeds": {
        "href": "https://frontier.testnet.digitalbits.io/transactions?order=desc&cursor=4010137949835264"
      },
      "transaction": {
        "href": "https://frontier.testnet.digitalbits.io/transactions/098824f8a801f5da220363a9039c2f253108d00305217c6360f125699324f2a8"
      }
    },
    "id": "098824f8a801f5da220363a9039c2f253108d00305217c6360f125699324f2a8",
    "paging_token": "4010137949835264",
    "successful": true,
    "hash": "098824f8a801f5da220363a9039c2f253108d00305217c6360f125699324f2a8",
    "created_at": "2021-06-14T18:45:46Z",
    "source_account": "GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT",
    "source_account_sequence": "4007749948014593",
    "fee_account": "GDXSEV5IUY6MHLHIYELNAKLOP626CHGDQHWZKJCLFP3OFEUMXFUFXKKT",
    "fee_charged": "100",
    "max_fee": "100",
    "operation_count": 1,
    "envelope_xdr": "AAAAAgAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAAGQADj0HAAAAAQAAAAEAAAAAAAAAAAAAAABgx6Q5AAAAAAAAAAEAAAAAAAAAAAAAAAAoxJqWgAv12W1ImfaBmhvpzYWklAFqE0R5PbhlzZXdbAAAAAJUC+QAAAAAAAAAAAGMuWhbAAAAQNQZ5/WwASuo38LGmDyj7A8QfUAdoOcjy0CNW/A8A2CGr/UcPIJ7ZatDWujib4eeDJWFopFQDYeTxs+lk5rpCQ4=",
    "result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=",
    "result_meta_xdr": "AAAAAgAAAAIAAAADAA4/MwAAAAAAAAAA7yJXqKY8w6zowRbQKW5/teEcw4HtlSRLK/bikoy5aFsAAAAXSHbnnAAOPQcAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAA4/MwAAAAAAAAAA7yJXqKY8w6zowRbQKW5/teEcw4HtlSRLK/bikoy5aFsAAAAXSHbnnAAOPQcAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMADj8zAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABdIduecAA49BwAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj8zAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABT0awOcAA49BwAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAADj8zAAAAAAAAAAAoxJqWgAv12W1ImfaBmhvpzYWklAFqE0R5PbhlzZXdbAAAAAJUC+QAAA4/MwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAA=",
    "fee_meta_xdr": "AAAABAAAAAMADj0HAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABdIdugAAA49BwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj8zAAAAAAAAAADvIleopjzDrOjBFtApbn+14RzDge2VJEsr9uKSjLloWwAAABdIduecAA49BwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMADj0HAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAxdoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEADj8zAAAAAAAAAAC300+A8SGiACMZeKQTbc3s0U6aNTBLD14/5rrFIEl/hAAAAAAAAxfMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
    "memo_type": "none",
    "signatures": [
      "1Bnn9bABK6jfwsaYPKPsDxB9QB2g5yPLQI1b8DwDYIav9Rw8gntlq0Na6OJvh54MlYWikVANh5PGz6WTmukJDg=="
    ],
    "valid_after": "1970-01-01T00:00:00Z",
    "valid_before": "2021-06-14T18:47:21Z",
    "ledger_attr": 933683
  }
]

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
