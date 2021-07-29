---
title: All Ledgers
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=ledgers&endpoint=all
---

This endpoint represents all [ledgers](../resources/ledger.md).
This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to get notifications as ledgers are closed by the DigitalBits network.
If called in streaming mode Frontier will start at the earliest known ledger unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream ledgers created since your request time.

## Request

```
GET /ledgers{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | `1623820974` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
# Retrieve the 200 latest ledgers, ordered chronologically
curl "https://frontier.testnet.digitalbits.io/ledgers?limit=200&order=desc"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.ledgers()
  .call()
  .then(function (ledgerResult) {
    // page 1
    console.log(JSON.stringify(ledgerResult.records))
  })
  .catch(function(err) {
    console.log(err)
  })

```


### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var ledgerHandler = function (ledgerResponse) {
  console.log(ledgerResponse);
};

var es = server.ledgers()
  .cursor('now')
  .stream({
    onmessage: ledgerHandler
})
```

## Response

This endpoint responds with a list of ledgers.  See [ledger resource](../resources/ledger.md) for reference.

### Example Response

```json
[
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/192"
      },
      "transactions": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/192/transactions{?cursor,limit,order}",
        "templated": true
      },
      "operations": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/192/operations{?cursor,limit,order}",
        "templated": true
      },
      "payments": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/192/payments{?cursor,limit,order}",
        "templated": true
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/192/effects{?cursor,limit,order}",
        "templated": true
      }
    },
    "id": "97b39c3900e7f30cfca8e9de957969480bf2639d6bbae423433dde84ed6ee8bf",
    "paging_token": "824633720832",
    "hash": "97b39c3900e7f30cfca8e9de957969480bf2639d6bbae423433dde84ed6ee8bf",
    "prev_hash": "e575d362aaaca24584aa71a69b3dde6acfb925e80b7ceb1b8d815699a2c5aff9",
    "sequence": 192,
    "successful_transaction_count": 0,
    "failed_transaction_count": 0,
    "operation_count": 0,
    "tx_set_operation_count": 0,
    "closed_at": "2021-04-13T13:49:26Z",
    "total_coins": "20000000000.0000000",
    "fee_pool": "0.0000000",
    "base_fee_in_nibbs": 100,
    "base_reserve_in_nibbs": 100000000,
    "max_tx_set_size": 100,
    "protocol_version": 15,
    "header_xdr": "AAAAD+V102KqrKJFhKpxpps93mrPuSXoC3zrG42BVpmixa/5jzaKZSZCfJ/sNjMNgnDzXqPoYJkcZ65S0XcyMrno44YAAAAAYHWhZgAAAAAAAAABAAAAAKqV7E8NgJZicfvA5TYMz5Vh5pluOP8ASgCgSd3q3d1NAAAAQPDoYJsZ8CmnZRBiAs7v9C/jV9f+K3QeQnnhvUTfrXukLeviM1jDD9l2Aecp1hcYqd2U+h3SfvSV2Dp1/Gn8zATfP2GYBKkv20BXGS3EPddI6neK3FK8SYzoBSTAFLgRGdUqNRu1cyy6DH9XVFSYcjGSBOXl16wbBekMjr0XoBQuAAAAwALGivC7FAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABkBfXhAAAAAGRcWfjB1FX6pk2n/C9G5i8Z102VUUTA+VRumlGGMdOMogAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
  },
  {
    "_links": {
      "self": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/193"
      },
      "transactions": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/193/transactions{?cursor,limit,order}",
        "templated": true
      },
      "operations": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/193/operations{?cursor,limit,order}",
        "templated": true
      },
      "payments": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/193/payments{?cursor,limit,order}",
        "templated": true
      },
      "effects": {
        "href": "https://frontier.testnet.digitalbits.io/ledgers/193/effects{?cursor,limit,order}",
        "templated": true
      }
    },
    "id": "4b7a21fd6842e03cae999819cfcaecfd5ed37943f2a57bd2c94e932ee89a0e51",
    "paging_token": "828928688128",
    "hash": "4b7a21fd6842e03cae999819cfcaecfd5ed37943f2a57bd2c94e932ee89a0e51",
    "prev_hash": "97b39c3900e7f30cfca8e9de957969480bf2639d6bbae423433dde84ed6ee8bf",
    "sequence": 193,
    "successful_transaction_count": 0,
    "failed_transaction_count": 0,
    "operation_count": 0,
    "tx_set_operation_count": 0,
    "closed_at": "2021-04-13T13:49:31Z",
    "total_coins": "20000000000.0000000",
    "fee_pool": "0.0000000",
    "base_fee_in_nibbs": 100,
    "base_reserve_in_nibbs": 100000000,
    "max_tx_set_size": 100,
    "protocol_version": 15,
    "header_xdr": "AAAAD5eznDkA5/MM/Kjp3pV5aUgL8mOda7rkI0M93oTtbui/gS87y+GxiWBWnRoNrI2VG2ZVP8hBeU3OuBycJi1+YSYAAAAAYHWhawAAAAAAAAABAAAAAKvZnpGcHWYsIfQkvokpnA88t6aedQMkQ3LW/icyV30jAAAAQP0ccgtyAsmeI9IvnFd+e611NT5ymafDtoDPtL+nVNimWlsiBGE/33wZkUi5kISs+/z2K7PAN4J0EIeVBxrOMA3fP2GYBKkv20BXGS3EPddI6neK3FK8SYzoBSTAFLgRGdUqNRu1cyy6DH9XVFSYcjGSBOXl16wbBekMjr0XoBQuAAAAwQLGivC7FAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABkBfXhAAAAAGRcWfjB1FX6pk2n/C9G5i8Z102VUUTA+VRumlGGMdOMogAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
  }
]

```

### Example Streaming Event

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/958657"
    },
    "transactions": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/958657/transactions{?cursor,limit,order}",
      "templated": true
    },
    "operations": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/958657/operations{?cursor,limit,order}",
      "templated": true
    },
    "payments": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/958657/payments{?cursor,limit,order}",
      "templated": true
    },
    "effects": {
      "href": "https://frontier.testnet.digitalbits.io/ledgers/958657/effects{?cursor,limit,order}",
      "templated": true
    }
  },
  "id": "d2b742d31acf6af9e41da960e74199da8f128e88ca55aac6cded14cf5fcec566",
  "paging_token": "4117400463081472",
  "hash": "d2b742d31acf6af9e41da960e74199da8f128e88ca55aac6cded14cf5fcec566",
  "prev_hash": "0430688a5394cbd3bdb20ace3fd46371df9821c4ebfc96c333783c26abfb2476",
  "sequence": 958657,
  "successful_transaction_count": 0,
  "failed_transaction_count": 0,
  "operation_count": 0,
  "tx_set_operation_count": 0,
  "closed_at": "2021-06-16T10:33:00Z",
  "total_coins": "20000000000.0000000",
  "fee_pool": "0.0207200",
  "base_fee_in_nibbs": 100,
  "base_reserve_in_nibbs": 100000000,
  "max_tx_set_size": 100,
  "protocol_version": 15,
  "header_xdr": "AAAADwQwaIpTlMvTvbIKzj/UY3HfmCHE6/yWwzN4PCar+yR28aQM7p8RkbDHmZP9TX4tU51fI3fmhaGp6f+ttaQC8jMAAAAAYMnTXAAAAAAAAAABAAAAAKvZnpGcHWYsIfQkvokpnA88t6aedQMkQ3LW/icyV30jAAAAQFYZQ3XhVMuHn6Myb3k6+8tklb2h3K1GbJF0KAlQTJJUG1isziJPXNnuy/dlujVhSTSU7vu/j1d374WfhzGnkgzfP2GYBKkv20BXGS3EPddI6neK3FK8SYzoBSTAFLgRGdFxiUWEuBYBeP+5fiimEr9ZxcFFuRh8iTRJ4wYjCJkTAA6gwQLGivC7FAAAAAAAAAADKWAAAAAAAAAAAAAAAAAAAABkBfXhAAAAAGTRcYlFhLgWAXj/uX4ophK/WcXBRbkYfIk0SeMGIwiZE7GnoJtIrqPU36tDFU4XORBgsCvIi04GG/A0tVIWclCY2pL7Nkua71s2zrhLvP2xk17wI1QdTs2NbP8p4hUvqO96TzEzCTu1IfbrP9QD0x0cN77mrkt2Hhi4BP6sYQcbDgAAAAA="
}
```

## Errors

- The [standard errors](../errors.md#standard-errors).
