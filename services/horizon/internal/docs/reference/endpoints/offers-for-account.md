---
title: Offers for Account
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=offers&endpoint=for_account
---

People on the Stellar network can make [offers](../resources/offer.md) to buy or sell assets.  This endpoint represents all the offers a particular account makes.
This endpoint can also be used in [streaming](../streaming.md) mode so it is possible to use it to listen as offers are processed in the Stellar network.
If called in streaming mode Horizon will start at the earliest known offer unless a `cursor` is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to only stream offers created since your request time.

## Request

```
GET /accounts/{account}/offers{?cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account` | required, string | Account ID | `GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR` |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. | `12884905984` |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/accounts/GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR/offers"
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.offers('accounts', 'GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR')
  .call()
  .then(function (offerResult) {
    console.log(offerResult);
  })
  .catch(function (err) {
    console.error(err);
  })
```

### JavaScript Streaming Example

```javascript
var StellarSdk = require('stellar-sdk')
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

var offerHandler = function (offerResponse) {
    console.log(offerResponse);
};

var es = server.offers('accounts', 'GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR')
    .stream({
        onmessage: offerHandler
    })
```

## Response

The list of offers.

### Example Response

```js
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR/offers?cursor=\u0026limit=10\u0026order=asc"
    },
    "next": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR/offers?cursor=8\u0026limit=10\u0026order=asc"
    },
    "prev": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR/offers?cursor=8\u0026limit=10\u0026order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://horizon-testnet.stellar.org/offers/8"
          },
          "offer_maker": {
            "href": "https://horizon-testnet.stellar.org/accounts/GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR"
          }
        },
        "id": 8,
        "paging_token": "8",
        "seller": "GBYTR4MC5JAX4ALGUBJD7EIKZVM7CUGWKXIUJMRSMK573XH2O7VAK3SR",
        "selling": {
          "asset_type": "credit_alphanum4",
          "asset_code": "BTC",
          "asset_issuer": "GB6FN4C7ZLWKENAOZDLZOQHNIOK4RDMV6EKLR53LWCHEBR6LVXOEKDZH"
        },
        "buying": {
          "asset_type": "native"
        },
        "amount": "8.0000000",
        "price_r": {
          "n": 1,
          "d": 1
        },
        "price": "1.0000000"
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard_Errors).
