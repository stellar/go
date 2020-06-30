---
title: Metrics
---

The metrics endpoint returns a host of useful data points for monitoring the health of the underlying Horizon process. 

Since Horizon 1.0.0 this endpoint is not part of the public API. It's available in the internal server (listening on the internal port set via `ADMIN_PORT` env variable or `--admin-port` CLI param).

## Request

```
GET /metrics
```

### curl Example Request

```sh
curl "https://horizon-testnet.stellar.org/metrics"
```


## Response

The `/metrics` endpoint returns a typical json response. Below, each section of related data points are grouped together and annotated (***note**: this endpoint returns ALL this data in one respons*e).

#### Links


|  Attribute   | Endpoint                                                                                           | Description                                                
|--------------|---------------------------------------------------------------------------------------------------|------------------------------------------------------------
| self  | `/metrics`| Link to self. |


##### *Example Response:*
```shell
"_links": {
  "self": {
    "href": "/metrics"
  }
}
```

#### Goroutines

Horizon utilizes Go's built in concurrency primitives ([goroutines](https://gobyexample.com/goroutines) and [channels](https://gobyexample.com/channels)). This metric monitors the number of currently running goroutines on this Horizon's process.

##### *Example Response:*
```shell
"goroutines": {
  "value": 3193
},
```

#### History

Horizon maintains its own database (postgres), a verbose and user friendly account of activity on the Stellar network.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| elder_ledger     | The sequence number of the oldest ledger recorded in Horizon's database. |
| latest_ledger    | The sequence number of the youngest (most recent) ledger recorded in Horizon's database.  |
| open_connections | The number of open connections to the Horizon database. |

##### *Example Response:*
```shell
"history.elder_ledger": {
  "value": 1
},
"history.latest_ledger": {
  "value": 19203710
},
"history.open_connections": {
  "value": 4
},
```

#### Ingester
Ingester represents metrics specific to Horizon's [ingestion](https://github.com/stellar/go/blob/master/services/horizon/internal/docs/reference/admin.md#ingesting-stellar-core-data) process, or the process by which Horizon consumes transaction results from a connected Stellar Core instance.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| clear_ledger |  The count and rate of clearing (per ledger) for this Horizon process.  |
| ingest_ledger | The count and rate of ingestion (per ledger)  for this Horizon process. |

These metrics contain useful [sub metrics](#sub-metrics).

```shell
"ingester.clear_ledger": {
  "15m.rate": 0,
  "1m.rate": 0,
  "5m.rate": 0,
  "75%": 0,
  "95%": 0,
  "99%": 0,
  "99.9%": 0,
  "count": 0,
  "max": 0,
  "mean": 0,
  "mean.rate": 0,
  "median": 0,
  "min": 0,
  "stddev": 0
},
"ingester.ingest_ledger": {
  "15m.rate": 0.20015746469980858,
  "1m.rate": 0.20369424731639432,
  "5m.rate": 0.20048643236880492,
  "75%": 13843204,
  "95%": 33225286.699999996,
  "99%": 55083311.51000008,
  "99.9%": 169331014.0600002,
  "count": 73796,
  "max": 171032904,
  "mean": 9263325.741245136,
  "mean.rate": 0.1999594297254709,
  "median": 3646103,
  "min": 17686,
  "stddev": 13151784.696390135
},
```

#### Logging

Horizon utilizes the standard `debug`, `error`, etc. levels of logging. This metric outputs stats for each level of log message produced, useful for a high-level monitoring of "is my Horizon instance functioning properly?" In order of increasing severity:

* debug
* info
* warning
* error
* panic

These metrics contain useful [sub metrics](#sub-metrics).

##### *Example Response:*
```shell
"logging.debug": {
  "15m.rate": 0,
  "1m.rate": 0,
  "5m.rate": 0,
  "count": 0,
  "mean.rate": 0
},
"logging.error": {
  "15m.rate": 1.2400751801386238e-53,
  "1m.rate": 2.964393875e-314,
  "5m.rate": 4.6339748288133364e-153,
  "count": 10,
  "mean.rate": 0.000027096232281212114
},
"logging.info": {
  "15m.rate": 236.778108287207,
  "1m.rate": 244.72112725997695,
  "5m.rate": 241.17582109786107,
  "count": 91818969,
  "mean.rate": 248.79427989406037
},
"logging.panic": {
  "15m.rate": 0,
  "1m.rate": 0,
  "5m.rate": 0,
  "count": 0,
  "mean.rate": 0
},
"logging.warning": {
  "15m.rate": 0,
  "1m.rate": 0,
  "5m.rate": 0,
  "count": 0,
  "mean.rate": 0
},
```

#### Requests

Requests represents an overview of Horizon's incoming traffic.

These metrics contain useful [sub metrics](#sub-metrics).

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| failed | Failed requests are those that return a status code in [400, 600). |
| succeeded | Successful requests are those that return a status code in [200, 400). |
| total | Total number of received requests.  |

##### *Example Response:*
```shell
"requests.failed": {
  "15m.rate": 22.419448554871114,
  "1m.rate": 33.09974656989629,
  "5m.rate": 26.745898323150673,
  "count": 8998213,
  "mean.rate": 24.38172358144542
},
"requests.succeeded": {
  "15m.rate": 76.81566860149793,
  "1m.rate": 78.85014329639597,
  "5m.rate": 77.96890869538107,
  "count": 29565299,
  "mean.rate": 80.11065378819903
},
"requests.total": {
  "15m.rate": 118.62183027832366,
  "1m.rate": 122.02995392023216,
  "5m.rate": 121.08103024256366,
  "75%": 8460442211,
  "95%": 59512053388.35,
  "99%": 59967007994.07,
  "99.9%": 121499506095.66623,
  "count": 45873213,
  "max": 123330396447,
  "mean": 7448679208.666343,
  "mean.rate": 124.2988641791271,
  "median": 13832639.5,
  "min": 20264,
  "stddev": 16825540184.214636
},
```

#### Stellar Core
As noted above, Horizon relies on Stellar Core to stay in sync with the Stellar network. These metrics are specific to the underlying Stellar Core instance.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| latest_ledger    | The sequence number of the latest (most recent) ledger recorded in Stellar Core's database.  |
| open_connections | The number of open connections to the Stellar Core postgres database.  |

##### *Example Response:*
```shell
"stellar_core.latest_ledger": {
  "value": 19203710
}
```

#### Transaction Submission

Horizon does not submit transactions directly to the Stellar network. Instead, it sequences transactions and sends the base64 encoded, XDR serialized blob to its connected Stellar Core instance. 

##### Horizon Transaction Sequencing and Submission

The following is a simplified version of the transaction submission process that glosses over the finer details. To dive deeper, check out the [source code](https://github.com/stellar/go/tree/master/services/horizon/internal/txsub).

Horizon's sequencing mechanism consists of a [manager](https://github.com/stellar/go/blob/master/services/horizon/internal/txsub/sequence/manager.go) that keeps track of [submission queues](https://github.com/stellar/go/blob/master/services/horizon/internal/txsub/sequence/queue.go) for a set of addresses. A submission queue is a  priority queue, prioritized by minimum transaction sequence number, that holds a set of pending transactions for an account. A pending transaction is represented as an object with a sequence number and a channel. Periodically, this queue is updated, popping off finished transactions, sending down the transaction's channel a successful/failure response.

These metrics contain useful [sub metrics](#sub-metrics).


|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| buffered | The count of submissions buffered behind this Horizon's submission queue.  |
| failed | The rate of failed transactions that have been submitted to this Horizon.  |
| open |The count of "open" submissions (i.e.) submissions whose transactions haven't been confirmed successful or failed.  |
| succeeded | The rate of successful transactions that have been submitted to this Horizon.  |
| total | Both the rate and count of all transactions submitted to this Horizon. |

##### *Example Response:*
```shell
"txsub.buffered": {
  "value": 0
},
"txsub.failed": {
  "15m.rate": 0.02617642409672995,
  "1m.rate": 0.030745796597772223,
  "5m.rate": 0.02768989245087351,
  "count": 8091,
  "mean.rate": 0.021923569654356184
},
"txsub.open": {
  "value": 1
},
"txsub.succeeded": {
  "15m.rate": 0.4530627633404272,
  "1m.rate": 0.5703741067858975,
  "5m.rate": 0.4981318318429001,
  "count": 194145,
  "mean.rate": 0.5260599963594831
},
"txsub.total": {
  "15m.rate": 0.47923918743715727,
  "1m.rate": 0.6011199033836697,
  "5m.rate": 0.5258217242937737,
  "75%": 44230960,
  "95%": 93582192.49999997,
  "99%": 288280173.0500002,
  "99.9%": 1413075508.2000008,
  "count": 202236,
  "max": 1418386864,
  "mean": 40583370.680933855,
  "mean.rate": 0.5479835660141416,
  "median": 20234100.5,
  "min": 3936410,
  "stddev": 95227916.95851417
}
```

### Sub Metrics
Various sub metrics related to a certain metric's performance.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| `1m.rate`, `5min.rate`, `etc.` | The per-minute moving average rate of events per second at the given time interval.  |
| `75%`, `95%`, `etc.` | Counts at different percentiles.  |
| `count` | Sum total of a certain metric value.  |
| `max`, `mean`, `etc.` |  Common statistic calculations. |




