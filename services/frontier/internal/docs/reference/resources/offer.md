---
title: Offer
---

Accounts on the DigitalBits network can make to buy or sell assets.  Users can create offers with the [Manage Offer](https://github.com/xdbfoundation/docs/blob/master/guides/concepts/list-of-operations.md#manage-offer) operation.

Frontier only returns offers that belong to a particular account.  When it does, it uses the following format:

## Attributes
| Attribute            | Type                                                              |                                                                                                                          |
|----------------------|-------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| id                   | string                                                            | The ID of this offer.                                                                                                    |
| paging_token         | string                                                            | A [paging token](https://github.com/xdbfoundation/go/tree/master/services/frontier/internal/docs/reference/resources/page.md) suitable for use as a `cursor` parameter.                                                    |
| seller               | string                                                            | Account id of the account making this offer.                                                                             |
| selling              | [Asset](https://github.com/xdbfoundation/docs/blob/master/guides/concepts/assets.md) | The Asset this offer wants to sell.                                                                                      |
| buying               | [Asset](https://github.com/xdbfoundation/docs/blob/master/guides/concepts/assets.md) | The Asset this offer wants to buy.                                                                                       |
| amount               | string                                                            | The amount of `selling` the account making this offer is willing to sell.                                                |
| price_r              | object                                                            | An object of a number numerator and number denominator that represent the buy and sell price of the currencies on offer. |
| price                | string                                                            | How many units of `buying` it takes to get 1 unit of `selling`. A number representing the decimal form of `price_r`.     |
| last_modified_ledger | integer                                                           | sequence number for the latest ledger in which this offer was modified.                                                  |
| last_modified_time   | string                                                            | An ISO 8601 formatted string of last modification time.                                                                  |

#### Price_r Object
Price_r is a more precise representation of a bid/ask offer.

| Attribute | Type   |                  |
|-----------|--------|------------------|
| n         | number | The numerator.   |
| d         | number | The denominator. |

Thus to get price you would take n / d.



## Links
| rel    | Example                                  | Description                                             | `templated` |
|--------|------------------------------------------|---------------------------------------------------------|-------------|
| seller | `/accounts/{seller}?cursor,limit,order}` | Link to details about the account that made this offer. | true        |

## Example

```json
{
  "_links": {
    "self": {
      "href": "https://frontier.testnet.digitalbits.io/offers/2611"
    },
    "offer_maker": {
      "href": "https://frontier.testnet.digitalbits.io/accounts/GDG3NOK5YI7A4FCBHE6SKI4L65R7UPRBZUZVBT44IBTQBWGUSTJDDKBQ"
    }
  },
  "id": "2611",
  "paging_token": "2611",
  "seller": "GDG3NOK5YI7A4FCBHE6SKI4L65R7UPRBZUZVBT44IBTQBWGUSTJDDKBQ",
  "selling": {
    "asset_type": "credit_alphanum12",
    "asset_code": "USD",
    "asset_issuer": "GCL3BJDFYQ2KAV7ARC4YCTERNJFOBOBQXSG556TX4YMOPKGEDV5K6LCQ"
  },
  "buying": {
    "asset_type": "native"
  },
  "amount": "1.0000000",
  "price_r": {
    "n": 1463518003,
    "d": 25041627
  },
  "price": "58.4434072",
  "last_modified_ledger": 196458,
  "last_modified_time": "2020-02-10T18:51:42Z"
}
```

## Endpoints

| Resource                                             | Type       | Resource URI Template          |
|------------------------------------------------------|------------|--------------------------------|
| [Offers](https://github.com/xdbfoundation/go/tree/master/services/frontier/internal/docs/reference/endpoints/offers.md)                     | Collection | `/offers`                      |
| [Account Offers](https://github.com/xdbfoundation/go/tree/master/services/frontier/internal/docs/reference/endpoints/offers-for-account.md) | Collection | `/accounts/:account_id/offers` |
| [Offers Details](https://github.com/xdbfoundation/go/tree/master/services/frontier/internal/docs/reference/endpoints/offer-details.md)      | Single     | `/offers/:offer_id`            |
