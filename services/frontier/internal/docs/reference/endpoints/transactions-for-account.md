This endpoint represents successful [transactions](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/resources/transaction) that affected a
given [account](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/resources/account).  This endpoint can also be used in
[streaming](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/streaming) mode so it is possible to use it to listen for new transactions that
affect a given account as they get made in the DigitalBits network.

If called in streaming mode Frontier will start at the earliest known transaction unless a `cursor`
is set. In that case it will start from the `cursor`. You can also set `cursor` value to `now` to
only stream transaction created since your request time.

## Request

```
GET /accounts/{account_id}/transactions{?cursor,limit,order,include_failed}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `account_id` | required, string | ID of an account | GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K |
| `?cursor` | optional, any, default _null_ | A paging token, specifying where to start returning records from. When streaming this can be set to `now` to stream object created since your request time. | 12884905984 |
| `?order`  | optional, string, default `asc` | The order in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |
| `?include_failed` | optional, bool, default: `false` | Set to `true` to include failed transactions in results. | `true` |

### curl Example Request

```sh
curl "https://frontier.testnet.digitalbits.io/accounts/GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K/transactions?limit=1"
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

server.transactions()
  .forAccount("GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K")
  .call()
  .then(function (accountResult) {
    console.log(accountResult);
  })
  .catch(function (err) {
    console.error(err);
  })
```

### JavaScript Streaming Example

```javascript
var DigitalBitsSdk = require('digitalbits-sdk')
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var txHandler = function (txResponse) {
  console.log(txResponse);
};

var es = server.transactions()
  .forAccount("GBS43BF24ENNS3KPACUZVKK2VYPOZVBQO2CISGZ777RYGOPYC2FT6S3K")
  .cursor('now')
  .stream({
    onmessage: txHandler
  })
```

## Response

This endpoint responds with a list of transactions that changed a given account's state. See
[transaction resource](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/resources/transaction) for reference.

### Example Response
```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF/payments?cursor=&limit=10&order=asc"
    },
    "next": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF/payments?cursor=2714719978786817&limit=10&order=asc"
    },
    "prev": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF/payments?cursor=1919197546291201&limit=10&order=desc"
    }
  },
  "_embedded": {
    "records": [
      {
        "_links": {
          "self": {
            "href": "https://frontier.testnet.digitalbits.io/operations/1919197546291201"
          },
          "transaction": {
            "href": "https://frontier.testnet.digitalbits.io/transactions/7e2050abc676003efc3eaadd623c927f753b7a6c37f50864bf284f4e1510d088"
          },
          "effects": {
            "href": "https://frontier.testnet.digitalbits.io/operations/1919197546291201/effects"
          },
          "succeeds": {
            "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=1919197546291201"
          },
          "precedes": {
            "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=1919197546291201"
          }
        },
        "id": "1919197546291201",
        "paging_token": "1919197546291201",
        "transaction_successful": true,
        "source_account": "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR",
        "type": "create_account",
        "type_i": 0,
        "created_at": "2019-03-25T22:43:38Z",
        "transaction_hash": "7e2050abc676003efc3eaadd623c927f753b7a6c37f50864bf284f4e1510d088",
        "starting_balance": "10000.0000000",
        "funder": "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR",
        "account": "GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF"
      },
      {
        "_links": {
          "self": {
            "href": "https://frontier.testnet.digitalbits.io/operations/2714719978786817"
          },
          "transaction": {
            "href": "https://frontier.testnet.digitalbits.io/transactions/7cea6abe90654578b42ee696e823187d89d91daa157a1077b542ee7c77413ce3"
          },
          "effects": {
            "href": "https://frontier.testnet.digitalbits.io/operations/2714719978786817/effects"
          },
          "succeeds": {
            "href": "https://frontier.testnet.digitalbits.io/effects?order=desc&cursor=2714719978786817"
          },
          "precedes": {
            "href": "https://frontier.testnet.digitalbits.io/effects?order=asc&cursor=2714719978786817"
          }
        },
        "id": "2714719978786817",
        "paging_token": "2714719978786817",
        "transaction_successful": true,
        "source_account": "GAGLYFZJMN5HEULSTH5CIGPOPAVUYPG5YSWIYDJMAPIECYEBPM2TA3QR",
        "type": "payment",
        "type_i": 1,
        "created_at": "2019-04-05T23:07:42Z",
        "transaction_hash": "7cea6abe90654578b42ee696e823187d89d91daa157a1077b542ee7c77413ce3",
        "asset_type": "credit_alphanum4",
        "asset_code": "FOO",
        "asset_issuer": "GAGLYFZJMN5HEULSTH5CIGPOPAVUYPG5YSWIYDJMAPIECYEBPM2TA3QR",
        "from": "GAGLYFZJMN5HEULSTH5CIGPOPAVUYPG5YSWIYDJMAPIECYEBPM2TA3QR",
        "to": "GBYUUJHG6F4EPJGNLERINATVQLNDOFRUD7SGJZ26YZLG5PAYLG7XUSGF",
        "amount": "1000000.0000000"
      }
    ]
  }
}
```

## Possible Errors

- The [standard errors](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/errors#standard-errors).
- [not_found](https://developers.digitalbits.io/reference/go/services/frontier/internal/docs/reference/errors/not-found): A `not_found` error will be returned if there is no account whose ID matches the `account_id` argument.
