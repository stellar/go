---
title: All Assets
---

This endpoint represents all [assets](../resources/asset.md).
It will give you all the assets in the system along with various statistics about each.

### Notes
- The attribute `num_accounts` includes authorized trust lines only.

## Request

```
GET /assets{?asset_code,asset_issuer,cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?asset_code` | optional, string, default _null_ | Code of the Asset to filter by | `USD` |
| `?asset_issuer` | optional, string, default _null_ | Issuer of the Asset to filter by | `GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ` |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. | `1` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc", ordered by asset_code then by asset_issuer. | `asc` |
| `?limit` | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
# Retrieve the 200 assets, ordered alphabetically:
curl "https://frontier.testnet.digitalbits.io/assets?limit=200"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.assets()
  .call()
  .then(function (result) {
    console.log(JSON.stringify(result.records));
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

If called normally this endpoint responds with a [page](../resources/page.md) of assets.

### Example Response

```json
[
  {
    "_links": {
      "toml": {
        "href": ""
      }
    },
    "asset_type": "credit_alphanum4",
    "asset_code": "EUR",
    "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC",
    "paging_token": "EUR_GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC_credit_alphanum4",
    "amount": "15.0000000",
    "num_accounts": 2,
    "flags": {
      "auth_required": false,
      "auth_revocable": false,
      "auth_immutable": false
    }
  },
  {
    "_links": {
      "toml": {
        "href": ""
      }
    },
    "asset_type": "credit_alphanum4",
    "asset_code": "HUF",
    "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "paging_token": "HUF_GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK_credit_alphanum4",
    "amount": "50000.0000000",
    "num_accounts": 1,
    "flags": {
      "auth_required": false,
      "auth_revocable": false,
      "auth_immutable": false
    }
  },
  {
    "_links": {
      "toml": {
        "href": ""
      }
    },
    "asset_type": "credit_alphanum4",
    "asset_code": "UAH",
    "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK",
    "paging_token": "UAH_GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK_credit_alphanum4",
    "amount": "10.0000000",
    "num_accounts": 1,
    "flags": {
      "auth_required": false,
      "auth_revocable": false,
      "auth_immutable": false
    }
  },
  {
    "_links": {
      "toml": {
        "href": ""
      }
    },
    "asset_type": "credit_alphanum4",
    "asset_code": "USD",
    "asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ",
    "paging_token": "USD_GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ_credit_alphanum4",
    "amount": "10.0000000",
    "num_accounts": 1,
    "flags": {
      "auth_required": false,
      "auth_revocable": false,
      "auth_immutable": false
    }
  }
]
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
