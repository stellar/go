---
title: Account Details
clientData:
  laboratoryUrl: https://www.stellar.org/laboratory/#explorer?resource=accounts&endpoint=single
---

Returns information and links relating to a single [account](../resources/account.md).

The balances section in the returned JSON will also list all the [trust lines](https://www.stellar.org/developers/learn/concepts/assets.html) this account has set up. Note this will only return trustlines that have the necessary authorization to work. Meaning if an accountA trusts another accountB that has the [authorization required](https://www.stellar.org/developers/guides/concepts/accounts.html#flags) flag set the trustline wont show up until accountB [allows](https://www.stellar.org/developers/guides/concepts/list-of-operations.html#allow-trust) accountA to hold its assets.

## Request

```
GET /accounts/{account}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account` | required, string | Account ID | GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW  |

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW "
```

### JavaScript Example Request

```javascript
var StellarSdk = require('stellar-sdk');
var server = new StellarSdk.Server('https://horizon-testnet.stellar.org');

server.accounts()
  .accountId("GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW ")
  .call()
  .then(function (accountResult) {
    console.log(accountResult);
  })
  .catch(function (err) {
    console.error(err);
  })
```

## Response

This endpoint responds with the details of a single account for a given ID. See [account resource](../resources/account.md) for reference.

### Example Response
```json
{
  "_links": {
    "self": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW"
    },
    "transactions": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW/transactions{?cursor,limit,order}",
      "templated": true
    },
    "operations": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW/operations{?cursor,limit,order}",
      "templated": true
    },
    "payments": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW/payments{?cursor,limit,order}",
      "templated": true
    },
    "effects": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW/effects{?cursor,limit,order}",
      "templated": true
    },
    "offers": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW/offers{?cursor,limit,order}",
      "templated": true
    },
    "trades": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW/trades{?cursor,limit,order}",
      "templated": true
    },
    "data": {
      "href": "https://horizon-testnet.stellar.org/accounts/GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW/data/{key}",
      "templated": true
    }
  },
  "id": "GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW",
  "paging_token": "",
  "account_id": "GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW",
  "sequence": "43692723777044483",
  "subentry_count": 3,
  "thresholds": {
    "low_threshold": 0,
    "med_threshold": 0,
    "high_threshold": 0
  },
  "flags": {
    "auth_required": false,
    "auth_revocable": false
  },
  "balances": [
    {
      "balance": "9999.9999700",
      "asset_type": "native"
    }
  ],
  "signers": [
    {
      "public_key": "GDLEPBJBC2VSKJCLJB264F2WDK63X4NKOG774A3QWVH2U6PERGDPUCS4",
      "weight": 1,
      "key": "GDLEPBJBC2VSKJCLJB264F2WDK63X4NKOG774A3QWVH2U6PERGDPUCS4",
      "type": "ed25519_public_key"
    },
    {
      "public_key": "XCPNCUKYDHPMMH6TMHK73K5VP5A6ZTQ2L7Q74JR3TDANNFB3TMRS5OKG",
      "weight": 1,
      "key": "XCPNCUKYDHPMMH6TMHK73K5VP5A6ZTQ2L7Q74JR3TDANNFB3TMRS5OKG",
      "type": "sha256_hash"
    },
    {
      "public_key": "TABGGIW6EXOVOSNJ2O27U2DUX7RWHSRBGOKQLGYDTOXPANEX6LXBX7O7",
      "weight": 1,
      "key": "TABGGIW6EXOVOSNJ2O27U2DUX7RWHSRBGOKQLGYDTOXPANEX6LXBX7O7",
      "type": "preauth_tx"
    },
    {
      "public_key": "GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW",
      "weight": 1,
      "key": "GBWRID7MPYUDBTNQPEHUN4XOBVVDPJOHYXAVW3UTOD2RG7BDAY6O3PHW",
      "type": "ed25519_public_key"
    }
  ],
  "data": {
    
  }
}
```

## Possible Errors

- The [standard errors](../errors.md#Standard-Errors).
- [not_found](../errors/not-found.md): A `not_found` error will be returned if there is no account whose ID matches the `account` argument.
