---
title: Offer
---

Accounts on the Stellar network can make [offers](http://stellar.org/developers/learn/concepts/exchange.html) to buy or sell assets.  Users can create offers with the [Manage Offer](http://stellar.org/developers/learn/concepts/list-of-operations.html) operation.

Horizon only returns offers that belong to a particular account.  When it does, it uses the following format:

## Attributes
| Attribute    | Type             |                                                                                                                        |
|--------------|------------------|------------------------------------------------------------------------------------------------------------------------|
| id           | integer           | The ID of this offer. |
| paging_token | string           | A [paging token](./page.md) suitable for use as a `cursor` parameter.                                                                |
| seller      | string           | Account id of the account making this offer.                                                    |
| selling     | [Asset](http://stellar.org/developers/learn/concepts/assets.html)           | The Asset this offer wants to sell.                      |
| buying     | [Asset](http://stellar.org/developers/learn/concepts/assets.html) | The Asset this offer wants to buy. |
| amount | string | The amount of `selling` the account making this offer is willing to sell.|
| price_r | object | An object of a number numerator and number denominator that represent the buy and sell price of the currencies on offer.|
| price| string | How many units of `buying` it takes to get 1 unit of `selling`. A number representing the decimal form of `price_r`.|
| last_modified| string | An ISO 8601 formatted string of last modification time||

## Links
| rel          | Example                                                                                           | Description                                                | `templated` |
|--------------|---------------------------------------------------------------------------------------------------|------------------------------------------------------------|-------------|
| seller      | `/accounts/{seller}?cursor,limit,order}`      | Link to details about the account that made this offer. | true        |


## Endpoints

| Resource                 | Type       | Resource URI Template                |
|--------------------------|------------|--------------------------------------|
| [Account Offers](../offers-for-account.md)       | Collection | `/accounts/:account_id/offers`       |
