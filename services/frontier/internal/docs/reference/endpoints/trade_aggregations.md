---
title: Trade Aggregations
---

Trade Aggregations are catered specifically for developers of trading clients. They facilitate
efficient gathering of historical trade data. This is done by dividing a given time range into
segments and aggregating statistics, for a given asset pair (`base`, `counter`) over each of these
segments.

The duration of the segments is specified with the `resolution` parameter. The start and end of the
time range are given by `startTime` and `endTime` respectively, which are both rounded to the
nearest multiple of `resolution` since epoch.

The individual segments are also aligned with multiples of `resolution` since epoch. If you want to
change this alignment, the segments can be offset by specifying the `offset` parameter.


## Request

```
GET /trade_aggregations?base_asset_type={base_asset_type}&base_asset_code={base_asset_code}&base_asset_issuer={base_asset_issuer}&counter_asset_type={counter_asset_type}&counter_asset_code={counter_asset_code}&counter_asset_issuer={counter_asset_issuer}
```

### Arguments

| name | notes | description | example |
| ---- | ----- | ----------- | ------- |
| `start_time` | long | lower time boundary represented as millis since epoch | 1512689100000 |
| `end_time` | long | upper time boundary represented as millis since epoch | 1512775500000 |
| `resolution` | long | segment duration as millis. *Supported values are 1 minute (60000), 5 minutes (300000), 15 minutes (900000), 1 hour (3600000), 1 day (86400000) and 1 week (604800000).* | 300000 |
| `offset` | long | segments can be offset using this parameter. Expressed in milliseconds. Can only be used if the resolution is greater than 1 hour. *Value must be in whole hours, less than the provided resolution, and less than 24 hours.* | 3600000 (1 hour) |
| `base_asset_type` | string | Type of base asset | `native` |
| `base_asset_code` | string | Code of base asset, not required if type is `native` | `EUR` |
| `base_asset_issuer` | string | Issuer of base asset, not required if type is `native` | `GDCIQQY2UKVNLLWGIX74DMTEAFCMQKAKYUWPBO7PLTHIHRKSFZN7V2FC` |
| `counter_asset_type` | string | Type of counter asset  | `credit_alphanum4` |
| `counter_asset_code` | string | Code of counter asset, not required if type is `native` | `USD` |
| `counter_asset_issuer` | string | Issuer of counter asset, not required if type is `native` | `GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ` |
| `?order`  | optional, string, default `asc` | The order, in terms of timeline, in which to return rows, "asc" or "desc". | `asc` |
| `?limit`  | optional, number, default: `10` | Maximum number of records to return. | `200` |

### curl Example Request
```sh
curl https://frontier.testnet.digitalbits.io/trade_aggregations?base_asset_type=native&counter_asset_code=USD&counter_asset_issuer=GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ&counter_asset_type=credit_alphanum4&limit=200&order=asc&resolution=60000&start_time=1623920055000&end_time=1623937426000
```

### JavaScript Example Request

```javascript
var DigitalBitsSdk = require('xdb-digitalbits-sdk');
var server = new DigitalBitsSdk.Server('https://frontier.testnet.digitalbits.io');

var base = new DigitalBitsSdk.Asset.native();
var counter = new DigitalBitsSdk.Asset("USD", "GB4RZUSF3HZGCAKB3VBM2S7QOHHC5KTV3LLZXGBYR5ZO4B26CKHFZTSZ");
var startTime = 1623920055000;
var endTime = 1623937426000;
var resolution = 60000;
var offset = 0;

server.tradeAggregation(base, counter, startTime, endTime, resolution, offset)
  .call()
  .then(function (tradeAggregation) {
    console.log(JSON.stringify(tradeAggregation));
  })
  .catch(function (err) {
    console.log(err);
  })
```

## Response

A list of collected trade aggregations.

Note

- Segments that fit into the time range but have 0 trades in them, will not be included.
- Partial segments, in the beginning and end of the time range, will not be included. Thus if your
  start time is noon Wednesday, your end time is noon Thursday, and your resolution is one day, you will not receive back any data. Instead, you would want to either start at midnight Wednesday and midnight Thursday, or shorten the resolution interval to better cover your time frame.

### Example Response
```json
{
  "records": [
    {
      "timestamp": "1623930840000",
      "trade_count": "1",
      "base_volume": "1.0000000",
      "counter_volume": "1.0000000",
      "avg": "1.0000000",
      "high": "1.0000000",
      "high_r": {
        "N": 1,
        "D": 1
      },
      "low": "1.0000000",
      "low_r": {
        "N": 1,
        "D": 1
      },
      "open": "1.0000000",
      "open_r": {
        "N": 1,
        "D": 1
      },
      "close": "1.0000000",
      "close_r": {
        "N": 1,
        "D": 1
      }
    }
  ]
}

```

## Possible Errors

- The [standard errors](../errors.md#standard-errors).
