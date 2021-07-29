---
title: Strict Receive Payment Paths
clientData:
  laboratoryUrl: https://laboratory.livenet.digitalbits.io/#explorer?resource=paths&endpoint=all
---

The DigitalBits Network allows payments to be made across assets through _path payments_.  A path
payment specifies a series of assets to route a payment through, from source asset (the asset
debited from the payer) to destination asset (the asset credited to the payee).

A Path Payment Strict Receive allows a user to specify the *amount of the asset received*. The amount sent varies based on offers in the order books.  If you would like to search for a path specifying the amount to be sent, use the [Find Payment Paths (Strict Send)](./path-finding-strict-send.md).

A strict receive path search is specified using:

- The source account id or source assets.
- The asset and amount that the destination account should receive.

As part of the search, frontier will load a list of assets available to the source account id and
will find any payment paths from those source assets to the desired destination asset. The search's
amount parameter will be used to determine if a given path can satisfy a payment of the
desired amount.

## Request

```
GET /paths/strict-receive?source_account={sa}&destination_asset_type={at}&destination_asset_code={ac}&destination_asset_issuer={di}&destination_amount={amount}&destination_account={da}
```

## Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?source_account` | string | The sender's account id. Any returned path must use an asset that the sender has a trustline to. | `GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY` |
| `?source_assets` | string | A comma separated list of assets. Any returned path must use an asset included in this list | `USD:GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ,native` |
| `?destination_account` | string | The destination account that any returned path should use | `GCSYKECRGY6VEF4F4KBZEEPXLYDLUGNZFCCXWR7SNRADN3NYYK67GQKF` |
| `?destination_asset_type` | string | The type of the destination asset | `credit_alphanum4` |
| `?destination_asset_code` | required if `destination_asset_type` is not `native`, string | The destination asset code, if destination_asset_type is not "native" | `USD` |
| `?destination_asset_issuer` | required if `destination_asset_type` is not `native`, string | The issuer for the destination asset, if destination_asset_type is not "native" | `GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ` |
| `?destination_amount` | string | The amount, denominated in the destination asset, that any returned path should be able to satisfy | `10.1` |

The endpoint will not allow requests which provide both a `source_account` and a `source_assets` parameter. All requests must provide one or the other.
The assets in `source_assets` are expected to be encoded using the following format:

XDB should be represented as `"native"`. Issued assets should be represented as `"Code:IssuerAccountID"`. `"Code"` must consist of alphanumeric ASCII characters.


### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/paths/strict-receive?destination_account=GCSYKECRGY6VEF4F4KBZEEPXLYDLUGNZFCCXWR7SNRADN3NYYK67GQKF&source_assets=native&destination_asset_code=USD&destination_asset_type=credit_alphanum4&destination_asset_issuer=GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ&destination_amount=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var destination_asset = new DigitalBitsSdk.Asset('USD', 'GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ');
var destination_amount = "1";

server.strictReceivePaths([new DigitalBitsSdk.Asset.native()], destination_asset, destination_amount)
  .call()
  .then(function (pathResult) {
    console.log(JSON.stringify(pathResult.records));
  })
  .catch(function (err) {
    console.log(err)
  })
```

## Response

This endpoint responds with a page of path resources.  See [path resource](../resources/path.md) for reference.

### Example Response

```json
[
  {
    "source_asset_type": "native",
    "source_amount": "0.2500000",
    "destination_asset_type": "credit_alphanum4",
    "destination_asset_code": "USD",
    "destination_asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ",
    "destination_amount": "1.0000000",
    "path": [
      {
        "asset_type": "credit_alphanum4",
        "asset_code": "UAH",
        "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
      },
      {
        "asset_type": "credit_alphanum4",
        "asset_code": "EUR",
        "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC"
      }
    ]
  },
  {
    "source_asset_type": "native",
    "source_amount": "1.0000000",
    "destination_asset_type": "credit_alphanum4",
    "destination_asset_code": "USD",
    "destination_asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ",
    "destination_amount": "1.0000000",
    "path": []
  }
]
```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if no paths could be found to fulfill this payment request
