---
title: Trade Aggregation
---

A Trade Aggregation represents aggregated statistics on an asset pair (`base` and `counter`) for a specific time period.

## Attributes
| Attribute    | Type             |                                                                                                                        |
|--------------|------------------|------------------------------------------------------------------------------------------------------------------------|
| timestamp | string | start time for this trade_aggregation. Represented as milliseconds since epoch.|
| trade_count |  int | total number of trades aggregated.|
| base_volume | string | total volume of `base` asset.|
| counter_volume | string | total volume of `counter` asset.|
| avg | string | weighted average price of `counter` asset in terms of `base` asset.|
| high | string | highest price for this time period.|
| high_r | object | highest price for this time period as a rational number.|
| low | string | lowest price for this time period.|
| low_r | object | lowest price for this time period as a rational number.|
| open | string | price as seen on first trade aggregated.|
| open_r | object | price as seen on first trade aggregated as a rational number.|
| close | string | price as seen on last trade aggregated.|
| close_r | object | price as seen on last trade aggregated as a rational number.|


## Endpoints

| Resource                 | Type       | Resource URI Template                |
|--------------------------|------------|--------------------------------------|
| [Trade Aggregations](../endpoints/trade_aggregations.md)       | Collection | `/trade_aggregations?{orderbook_params}`       |
