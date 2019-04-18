# Ticker API Documentation

## Market (Ticker) Data
Provides trade data about each trade pair within the last 7-day period. Asset pairs that did not have any activity in the last 7 days are omitted from the response.

Assets from different issuers but with the same code are aggregated, so trades between, for instance:
- `native` and `BTC:GDT3ZKQZXXHDPJUKNHUMANMNIT4JWSUYXUGN7EQZDVXBO7NPNFVFPBAK`
- `native` and `BTC:GATEMHCCKCY67ZUCKTROYN24ZYT5GK4EQZ65JJLDHKHRUZI3EUEKMTCH`

are aggregated in the `XLM_BTC` pair.

### Response Fields

* `generated_at`: UNIX timestamp of when data was generated
* `name`: name of the trade pair
* `base_volume`: accumulated amount of base traded in the last 24h
* `counter_volume`: accumulated amount of counter traded in the last 24h
* `trade_count`: number of trades in the last 24h
* `open`: open price in the last 24h period
* `low`: lowest price in the last 24h
* `high`: highest price in the last 24h
* `change`: price difference between open and low in the last 24h
* `base_volume_7d`: accumulated amount of base traded in the last 7 days
* `counter_volume_7d`: accumulated amount of counter traded in the last 7 days
* `trade_count_7d`: number of trades in the last 7 days
* `open_7d`: open price in the last 7-day period
* `low_7d`: lowest price in the last 7 days
* `high_7d`: highest price in the last 7 days
* `change_7d`: price difference between open and low in the last 7 days
* `price`: (DEPRECATED) price of the most recent trade in this market
* `close`: price of the most recent trade in this market
* `close_time`: ledger close time of the most recent trade in this market

### Example
#### Endpoint
GET `https://ticker.stellar.org/markets.json`
#### Response (application/json)

```json
{
    "generated_at": 1555529876427,
    "pairs": [
        {
            "name": "ABDT_DOP",
            "base_volume": 2936.8559372,
            "counter_volume": 67288.46914140001,
            "trade_count": 48,
            "open": 0.043734167043589976,
            "low": 0.04348452989055152,
            "high": 0.044444444444444446,
            "change": -0.0001736265897799502,
            "base_volume_7d": 93793.14427990005,
            "counter_volume_7d": 1939718.2717202017,
            "trade_count_7d": 399,
            "open_7d": 0.049850448654037885,
            "low_7d": 0.04348452989055152,
            "high_7d": 0.05100629529387713,
            "change_7d": 0.0059426550206679585,
            "price": 0.043907793633369926,
            "close": 0.043907793633369926,
            "close_time": 1555484876000
        },
        {
            "name": "BTC_ETH",
            "base_volume": 0,
            "counter_volume": 0,
            "trade_count": 0,
            "open": 0,
            "low": 0,
            "high": 0,
            "change": 0,
            "base_volume_7d": 0.00001,
            "counter_volume_7d": 0.00025,
            "trade_count_7d": 1,
            "open_7d": 0,
            "low_7d": 0.04,
            "high_7d": 0.04,
            "change_7d": 0,
            "price": 0.04,
            "close": 0.04,
            "close_time": 1554943803000
        }
    ]
}
```
## Asset (Currency) Data
Lists all the valid assets within the Stellar network. The provided fields are based on the [Currency Documentation of SEP-0001](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0001.md#currency-documentation) and the [Asset fields from Horizon](https://www.stellar.org/developers/horizon/reference/resources/asset.html).
### Response Fields

* `generated_at`: UNIX timestamp of when data was generated
* `code`: code of the asset
* `issuer`: token issuer Stellar public key
* `type`: type of the asset (e.g. `native` or `credit_alphanum4`)
* `num_accounts`: the number of accounts that: 1) trust this asset and 2) where if the asset has the auth_required flag then the account is authorized to hold the asset.
* `auth_required`: an anchor must approve anyone who wants to hold its asset
* `auth_revocable`: an anchor can set the authorize flag of an existing trustline to freeze the assets held by an asset holder
* `amount`: number of units of credit issued
* `asset_controlled_by_domain`: whether the reported issuer domain controls the asset
* `is_asset_anchored`: whether the asset is anchored to another one
* `anchor_asset`: anchor asset associated (e.g. USDT is anchored to USD)
* `anchor_asset_type`: type of anchor asset
* `display_decimals`: preference for number of decimals to show when a client displays currency balance
* `name`: name of the token
* `desc`: description of the token
* `conditions`: conditions on token
* `fixed_number`: if the number of tokens issued will never change
* `max_number`: max number of tokens, if there will never be more than max_number tokens
* `is_unlimited`: if the number of tokens is dilutable at the issuer's discretion
* `redemption_instructions`: if anchored token, these are instructions to redeem the underlying asset from tokens.
* `collateral_addresses`: if this is an anchored crypto token, list of one or more public addresses that hold the assets for which you are issuing tokens.
* `collateral_address_signatures`: occasional collateral address signatures
* `countries`: countries in which the asset is available
* `status`: status of token
* `last_valid`: last the time the asset info was validated

### Example
#### Endpoint
GET `https://ticker.stellar.org/assets.json`

#### Response (application/json)

```json
{
    "generated_at": 1555536150576,
    "assets": [
        {
            "code": "AngelXYZ",
            "issuer": "GANZBUS4726LBT2CBJ3BGF3TP4NJT5MHZMI423NHMXFRWGO2DCBQEXYZ",
            "type": "credit_alphanum12",
            "num_accounts": 282,
            "auth_required": false,
            "auth_revocable": false,
            "amount": 4999999999.999953,
            "asset_controlled_by_domain": true,
            "anchor_asset": "",
            "anchor_asset_type": "",
            "display_decimals": 0,
            "name": "",
            "desc": "",
            "conditions": "",
            "is_asset_anchored": false,
            "fixed_number": 0,
            "max_number": 0,
            "is_unlimited": false,
            "redemption_instructions": "",
            "collateral_addresses": [],
            "collateral_address_signatures": [],
            "countries": "",
            "status": "",
            "last_valid": 1555509989002
        },
        {
            "code": "PUSH",
            "issuer": "GBB5TTFQE5KT3TEBCR7Z3FZR3R3WTVD654XL2KHKVONRIOBEI5UGOFQQ",
            "type": "credit_alphanum4",
            "num_accounts": 15,
            "auth_required": false,
            "auth_revocable": false,
            "amount": 1000000000,
            "asset_controlled_by_domain": true,
            "anchor_asset": "",
            "anchor_asset_type": "",
            "display_decimals": 2,
            "name": "Push",
            "desc": "1 PUSH token entitles you to access the push API.",
            "conditions": "Token used to access the PUSH api to send a push request to the stellar network.",
            "is_asset_anchored": false,
            "fixed_number": 0,
            "max_number": 0,
            "is_unlimited": false,
            "redemption_instructions": "",
            "collateral_addresses": [],
            "collateral_address_signatures": [],
            "countries": "",
            "status": "",
            "last_valid": 1555509990457
        }
    ]
}

```

## Orderbook
Orderbook data can be retrieved directly from Horizon. In order to retrieve `ask` and `bid` data, you have to provide the following parameters from the asset pairs:

- `selling_asset_type`: type of selling asset (e.g. `native`, `credit_alphanum4`)
- `selling_asset_code`: code of the selling asset. Omit if `selling_asset_type` = `native`
- `selling_asset_issuer`: selling asset's issuer ID. Omit if `selling_asset_type` = `native`
- `buying_asset_type`: type of buying asset (e.g. `native` or `credit_alphanum4`)
- `buying_asset_code`: code of the buying asset. Omit if `buying_asset_type` = `native`
- `buying_asset_issuer`: buying asset's issuer ID. Omit if `buying_asset_type` = `native`

The `type`, `code` and `issuer` parameters for any given asset can be found in the Ticker's `assets.json` endpoint described in the previous section.


Full documentation on Horizon's Orderbook endpoint can be found [here](https://www.stellar.org/developers/horizon/reference/endpoints/orderbook-details.html).

### Example
#### Endpoint
GET `https://horizon.stellar.org/order_book?selling_asset_type=native&buying_asset_type=credit_alphanum4&buying_asset_code=BTC&buying_asset_issuer=GATEMHCCKCY67ZUCKTROYN24ZYT5GK4EQZ65JJLDHKHRUZI3EUEKMTCH`

#### Response (application/json)
```json
{
    "bids": [
        {
            "price_r": {
                "n": 223,
                "d": 10000000
            },
            "price": "0.0000223",
            "amount": "0.0006261"
        },
        {
            "price_r": {
                "n": 16469,
                "d": 739692077
            },
            "price": "0.0000223",
            "amount": "0.0037850"
        },
        {
            "price_r": {
                "n": 16469,
                "d": 741750702
            },
            "price": "0.0000222",
            "amount": "0.0037745"
        },
        {
            "price_r": {
                "n": 111,
                "d": 5000000
            },
            "price": "0.0000222",
            "amount": "0.0040000"
        }
    ],
    "asks": [
        {
            "price_r": {
                "n": 7,
                "d": 312500
            },
            "price": "0.0000224",
            "amount": "150.8482143"
        },
        {
            "price_r": {
                "n": 9,
                "d": 400000
            },
            "price": "0.0000225",
            "amount": "348.4311112"
        },
        {
            "price_r": {
                "n": 113,
                "d": 5000000
            },
            "price": "0.0000226",
            "amount": "335.6238939"
        }
    ],
    "base": {
        "asset_type": "native"
    },
    "counter": {
        "asset_type": "credit_alphanum4",
        "asset_code": "BTC",
        "asset_issuer": "GATEMHCCKCY67ZUCKTROYN24ZYT5GK4EQZ65JJLDHKHRUZI3EUEKMTCH"
    }
}
```
