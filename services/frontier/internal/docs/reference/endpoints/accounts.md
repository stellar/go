---
title: Accounts
---

This endpoint allows filtering accounts who have a given `signer` or have a trustline to an `asset`. The result is a list of [accounts](../resources/account.md).

To find all accounts who are trustees to an asset, pass the query parameter `asset` using the canonical representation for an issued assets which is `Code:IssuerAccountID`. 

### Notes
- The default behavior when filtering by `asset` is to return accounts with `authorized` and `unauthorized` trustlines.

## Request

```
GET /accounts{?signer,asset,cursor,limit,order}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `?signer` | optional, string | Account ID | GD42RQNXTRIW6YR3E2HXV5T2AI27LBRHOERV2JIYNFMXOBA234SWLQQB |
| `?asset` | optional, string | An issued asset represented as "Code:IssuerAccountID". | `USD:GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ,native` |
| `?cursor` | optional, default _null_ | A paging token, specifying where to start returning records from. | `GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ` |
| `?order` | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit` | optional, number, default `10` | Maximum number of records to return. | `200` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts?signer=GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY"
```

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk'); 
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io'); 
server.accounts().forAsset("USD:GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ")
   .call() 
   .then(function (accountResult) { 
     console.log(JSON.stringify(accountResult)); 
   }) 
   .catch(function (err) { 
     console.error(err); 
   }) 

```

## Response

This endpoint responds with the details of all accounts matching the filters. See [account resource](../resources/account.md) for reference.

### Example Response

```json
{
  "records": [
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M"
        },
        "transactions": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/transactions{?cursor,limit,order}",
          "templated": true
        },
        "operations": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/operations{?cursor,limit,order}",
          "templated": true
        },
        "payments": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/payments{?cursor,limit,order}",
          "templated": true
        },
        "effects": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/effects{?cursor,limit,order}",
          "templated": true
        },
        "offers": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/offers{?cursor,limit,order}",
          "templated": true
        },
        "trades": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/trades{?cursor,limit,order}",
          "templated": true
        },
        "data": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M/data/{key}",
          "templated": true
        }
      },
      "id": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
      "account_id": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
      "sequence": "4109317334630410",
      "subentry_count": 6,
      "last_modified_ledger": 974547,
      "last_modified_time": "2021-06-17T11:54:15Z",
      "thresholds": {
        "low_threshold": 0,
        "med_threshold": 0,
        "high_threshold": 0
      },
      "flags": {
        "auth_required": false,
        "auth_revocable": false,
        "auth_immutable": false
      },
      "balances": [
        {
          "balance": "0.0000000",
          "limit": "922337203685.4775807",
          "buying_liabilities": "0.0000000",
          "selling_liabilities": "0.0000000",
          "last_modified_ledger": 971809,
          "is_authorized": true,
          "is_authorized_to_maintain_liabilities": true,
          "asset_type": "credit_alphanum4",
          "asset_code": "EUR",
          "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC"
        },
        {
          "balance": "10000.0000000",
          "limit": "922337203685.4775807",
          "buying_liabilities": "0.0000000",
          "selling_liabilities": "1490.0000000",
          "last_modified_ledger": 973175,
          "is_authorized": true,
          "is_authorized_to_maintain_liabilities": true,
          "asset_type": "credit_alphanum4",
          "asset_code": "UAH",
          "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
        },
        {
          "balance": "1.0000000",
          "limit": "922337203685.4775807",
          "buying_liabilities": "1000.0000000",
          "selling_liabilities": "0.0000000",
          "last_modified_ledger": 974547,
          "is_authorized": true,
          "is_authorized_to_maintain_liabilities": true,
          "asset_type": "credit_alphanum4",
          "asset_code": "USD",
          "asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ"
        },
        {
          "balance": "10003.9999000",
          "buying_liabilities": "395.0000000",
          "selling_liabilities": "0.0000000",
          "asset_type": "native"
        }
      ],
      "signers": [
        {
          "weight": 1,
          "key": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
          "type": "ed25519_public_key"
        }
      ],
      "num_sponsoring": 0,
      "num_sponsored": 0,
      "paging_token": "GCKY3VKRJDSRORRMHRDHA6IKRXMGSBRZE42P64AHX4NHVGB3Y224WM3M",
      "data_attr": {}
    },
    {
      "_links": {
        "self": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY"
        },
        "transactions": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/transactions{?cursor,limit,order}",
          "templated": true
        },
        "operations": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/operations{?cursor,limit,order}",
          "templated": true
        },
        "payments": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/payments{?cursor,limit,order}",
          "templated": true
        },
        "effects": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/effects{?cursor,limit,order}",
          "templated": true
        },
        "offers": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/offers{?cursor,limit,order}",
          "templated": true
        },
        "trades": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/trades{?cursor,limit,order}",
          "templated": true
        },
        "data": {
          "href": "https://frontier.testnet.digitalbits.io/accounts/GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY/data/{key}",
          "templated": true
        }
      },
      "id": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
      "account_id": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
      "sequence": "4113023891406861",
      "subentry_count": 9,
      "last_modified_ledger": 974547,
      "last_modified_time": "2021-06-17T11:54:15Z",
      "thresholds": {
        "low_threshold": 0,
        "med_threshold": 0,
        "high_threshold": 0
      },
      "flags": {
        "auth_required": false,
        "auth_revocable": false,
        "auth_immutable": false
      },
      "balances": [
        {
          "balance": "5005.0000000",
          "limit": "922337203685.4775807",
          "buying_liabilities": "46.0000000",
          "selling_liabilities": "0.0000000",
          "last_modified_ledger": 971390,
          "is_authorized": true,
          "is_authorized_to_maintain_liabilities": true,
          "asset_type": "credit_alphanum4",
          "asset_code": "EUR",
          "asset_issuer": "GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC"
        },
        {
          "balance": "50000.0000000",
          "limit": "922337203685.4775807",
          "buying_liabilities": "0.0000000",
          "selling_liabilities": "0.0000000",
          "last_modified_ledger": 958064,
          "is_authorized": true,
          "is_authorized_to_maintain_liabilities": true,
          "asset_type": "credit_alphanum4",
          "asset_code": "HUF",
          "asset_issuer": "GCHQ6AOZST6YPMROCQWPE3SVFY57FHPYC3WJGGSFCHOQ5HFZC5HSHQYK"
        },
        {
          "balance": "10009.0000000",
          "limit": "922337203685.4775807",
          "buying_liabilities": "0.0000000",
          "selling_liabilities": "1102.0000000",
          "last_modified_ledger": 974547,
          "is_authorized": true,
          "is_authorized_to_maintain_liabilities": true,
          "asset_type": "credit_alphanum4",
          "asset_code": "USD",
          "asset_issuer": "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ"
        },
        {
          "balance": "10000.9998700",
          "buying_liabilities": "1101.0000000",
          "selling_liabilities": "0.0000000",
          "asset_type": "native"
        }
      ],
      "signers": [
        {
          "weight": 1,
          "key": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
          "type": "ed25519_public_key"
        }
      ],
      "num_sponsoring": 0,
      "num_sponsored": 0,
      "paging_token": "GDFOHLMYCXVZD2CDXZLMW6W6TMU4YO27XFF2IBAFAV66MSTPDDSK2LAY",
      "data_attr": {
        "user-id": "WERCRm91bmRhdGlvbg=="
      }
    }
  ]
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
