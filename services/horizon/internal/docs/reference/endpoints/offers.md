---
title: Offers
---

People on the Stellar network can make [offers](../resources/offer.md) to buy or sell assets. This
endpoint represents all the current offers, allowing filtering by `seller`, `selling_asset` or `buying_asset`.

**Note**: This endpoint is still experimental and available only if Horizon is running the [new ingestion system](https://github.com/stellar/go/blob/master/services/horizon/internal/expingest/BETA_TESTING.md).

## Request

```
GET /offers{?selling_asset_type,selling_asset_issuer,selling_asset_code,buying_asset_type,buying_asset_issuer,buying_asset_code,seller,cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?seller` | optional, string | Account ID of the offer creator  | `GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36` |
| `?selling_asset_type` | required, string | Type of the Asset being sold | `native` |
| `?selling_asset_code` | required if `selling_asset_type` is not `native`, string | Code of the Asset being sold | `USD` |
| `?selling_asset_issuer` | required if `selling_asset_type` is not `native`, string | Account ID of the issuer of the Asset being sold | `GA2HGBJIJKI6O4XEM7CZWY5PS6GKSXL6D34ERAJYQSPYA6X6AI7HYW36` |
| `?buying_asset_type` | required, string | Type of the Asset being bought | `credit_alphanum4` |
| `?buying_asset_code` |   required if buying_asset_type is not `native`, string | Code of the Asset being bought | `BTC` |
| `?buying_asset_issuer` | required if buying_asset_type is not `native`, string | Account ID of the issuer of the Asset being bought | `GD6VWBXI6NY3AOOR55RLVQ4MNIDSXE5JSAVXUTF35FRRI72LYPI3WL6Z` |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. | `12884905984` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/offers{?selling_asset_type,selling_asset_issuer,selling_asset_code,buying_asset_type,buying_asset_issuer,buying_asset_code,seller,cursor,limit,order}"
```

<!-- ### JavaScript Example Request -->

<!-- ```javascript -->
<!-- var StellarSdk = require('stellar-sdk'); -->
<!-- var server = new StellarSdk.Server('https://horizon-testnet.stellar.org'); -->

<!-- server.offers('accounts', 'GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF') -->
<!--   .call() -->
<!--   .then(function (offerResult) { -->
<!--     console.log(offerResult); -->
<!--   }) -->
<!--   .catch(function (err) { -->
<!--     console.error(err); -->
<!--   }) -->
<!-- ``` -->

<!-- ### JavaScript Streaming Example -->

<!-- ```javascript -->
<!-- var StellarSdk = require('stellar-sdk') -->
<!-- var server = new StellarSdk.Server('https://horizon-testnet.stellar.org'); -->

<!-- var offerHandler = function (offerResponse) { -->
<!--   console.log(offerResponse); -->
<!-- }; -->

<!-- var es = server.offers('accounts', 'GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF') -->
<!--   .cursor('now') -->
<!--   .stream({ -->
<!--     onmessage: offerHandler -->
<!--   }) -->
<!-- ``` -->

## Response

The list of offers.

### Example Response

```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/offers?cursor=&limit=10&order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/offers?cursor=5443256&limit=10&order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/offers?cursor=5443256&limit=10&order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://horizon-testnet.stellar.org/offers/5443256"
          },
          "offer_maker": {
            "href": "https://horizon-testnet.stellar.org/
          }
        },
        "id": 5443256,
        "paging_token": "5443256",
        "seller": "GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF",
        "selling": {
          "asset_type": "native"
        },
        "buying": {
          "asset_type": "credit_alphanum4",
          "asset_code": "FOO",
          "asset_issuer": "GAGLYFZJMN5HEULSTH5CIGPOPAVUYPG5YSWIYDJMAPIECYEBPM2TA3QR"
        },
        "amount": "10.0000000",
        "price_r": {
          "n": 1,
          "d": 1
        },
        "price": "1.0000000",
        "last_modified_ledger": 694974,
        "last_modified_time": "2019-04-09T17:14:22Z"
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard_Errors).
