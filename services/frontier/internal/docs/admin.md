---
title: Frontier Administration Guide
---

## Frontier Administration Guide

Frontier is responsible for providing an HTTP API to data in the DigitalBits network. It ingests and re-serves the data produced by the digitalbits network in a form that is easier to consume than the performance-oriented data representations used by digitalbits-core.

This document describes how to administer a **production** Frontier instance. If you are just starting with Frontier and want to try it out, consider the [Quickstart Guide](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/quickstart.md) instead. For information about developing on the Frontier codebase, check out the [Development Guide](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/developing.md).

## Why run Frontier?

The DigitalBits Development Foundation runs two Frontier servers, one for the public network and one for the test network, free for anyone's use at https://frontier.livenet.digitalbits.io and https://frontier.testnet.digitalbits.io.  These servers should be fine for development and small scale projects, but it is not recommended that you use them for production services that need strong reliability.  By running Frontier within your own infrastructure provides a number of benefits:

  - Multiple instances can be run for redundancy and scalability.
  - Request rate limiting can be disabled.
  - Full operational control without dependency on the DigitalBits Development Foundations operations.

## Prerequisites

Frontier is dependent upon a digitalbits-core server.  Frontier needs access to both the SQL database and the HTTP API that is published by digitalbits-core. See [the administration guide](https://github.com/xdbfoundation/DigitalBits/blob/master/docs/software/admin.md) to learn how to set up and administer a digitalbits-core server.  Secondly, Frontier is dependent upon a postgres server, which it uses to store processed core data for ease of use. Frontier requires postgres version >= 9.5.

## Installing

To install Frontier, you have a choice: either downloading a [prebuilt release for your target architecture](https://cloudsmith.io/~xdb-foundation/repos/digitalbits-frontier/packages/) and operation system, or [building Frontier yourself](#building).  When either approach is complete, you will find yourself with a directory containing a file named `frontier`.  This file is a native binary.

After building or unpacking Frontier, you simply need to copy the native binary into a directory that is part of your PATH.  Most unix-like systems have `/usr/local/bin` in PATH by default, so unless you have a preference or know better, we recommend you copy the binary there.

To test the installation, simply run `frontier --help` from a terminal.  If the help for Frontier is displayed, your installation was successful. Note: some shells, such as zsh, cache PATH lookups.  You may need to clear your cache  (by using `rehash` in zsh, for example) before trying to run `frontier --help`.


## Building

Should you decide not to use one of our prebuilt releases, you may instead build Frontier from source.  To do so, you need to install some developer tools:

- A unix-like operating system with the common core commands (cp, tar, mkdir, bash, etc.)
- A compatible distribution of Go (Go 1.14 or later)
- [git](https://git-scm.com/)
- [mercurial](https://www.mercurial-scm.org/)

1. See the details in [README.md](https://github.com/xdbfoundation/go/blob/master/README.md#dependencies) for installing dependencies.
2. Compile the Frontier binary: `go install github.com/xdbfoundation/go/services/frontier`. You should see the `frontier` binary in `$GOPATH/bin`.
3. Add Go binaries to your PATH in your `bashrc` or equivalent, for easy access: `export PATH=${GOPATH//://bin:}/bin:$PATH`

Open a new terminal. Confirm everything worked by running `frontier --help` successfully.

Note:  Building directly on windows is not supported.


## Configuring

Frontier is configured using command line flags or environment variables.  To see the list of command line flags that are available (and their default values) for your version of Frontier, run:

`frontier --help`

As you will see if you run the command above, Frontier defines a large number of flags, however only three are required:

| flag                    | envvar                      | example                              |
|-------------------------|-----------------------------|--------------------------------------|
| `--db-url`              | `DATABASE_URL`              | postgres://localhost/frontier_testnet |
| `--digitalbits-core-db-url` | `DIGITALBITS_CORE_DATABASE_URL` | postgres://localhost/core_testnet    |
| `--digitalbits-core-url`    | `DIGITALBITS_CORE_URL`          | http://localhost:11626               |

`--db-url` specifies the Frontier database, and its value should be a valid [PostgreSQL Connection URI](https://www.postgresql.org/docs/9.6/libpq-connect.html#AEN46025).  `--digitalbits-core-db-url` specifies a digitalbits-core database which will be used to load data about the digitalbits ledger.  Finally, `--digitalbits-core-url` specifies the HTTP control port for an instance of digitalbits-core.  This URL should be associated with the digitalbits-core that is writing to the database at `--digitalbits-core-db-url`.

Specifying command line flags every time you invoke Frontier can be cumbersome, and so we recommend using environment variables.  There are many tools you can use to manage environment variables:  we recommend either [direnv](http://direnv.net/) or [dotenv](https://github.com/bkeepers/dotenv).



## Preparing the database

Before the Frontier server can be run, we must first prepare the Frontier database.  This database will be used for all of the information produced by Frontier, notably historical information about successful transactions that have occurred on the digitalbits network.

To prepare a database for Frontier's use, first you must ensure the database is blank.  It's easiest to simply create a new database on your postgres server specifically for Frontier's use.  Next you must install the schema by running `frontier db init`.  Remember to use the appropriate command line flags or environment variables to configure Frontier as explained in [Configuring](#Configuring).  This command will log any errors that occur.

### Postgres configuration

It is recommended to set `random_page_cost=1` in Postgres configuration if you are using SSD storage. With this setting Query Planner will make a better use of indexes, especially for `JOIN` queries. We have noticed a huge speed improvement for some queries.

## Running

Once your Frontier database is configured, you're ready to run Frontier.  To run Frontier you simply run `frontier` or `frontier serve`, both of which start the HTTP server and start logging to standard out.  When run, you should see some output that similar to:

```
INFO[0000] Starting frontier on :8000                     pid=29013
```

The log line above announces that Frontier is ready to serve client requests. Note: the numbers shown above may be different for your installation.  Next we can confirm that Frontier is responding correctly by loading the root resource.  In the example above, that URL would be [http://127.0.0.1:8000/](http://127.0.0.1:8000/) and simply running 

```curl http://127.0.0.1:8000/```

 shows you that the root resource can be loaded correctly.

If you didn't set up a digitalbits-core yet, you may see an error like this:

```
ERRO[2019-05-06T16:21:14.126+08:00] Error getting core latest ledger err="get failed: pq: relation \"ledgerheaders\" does not exist"
```

Frontier requires a functional digitalbits-core. Go back and set up digitalbits-core as described in the admin guide. In particular, you need to initialise the database as [described here](https://github.com/xdbfoundation/DigitalBits/blob/master/docs/software/admin.md#database-and-local-state).

## Ingesting live digitalbits-core data

Frontier provides most of its utility through ingested data.  Your Frontier server can be configured to listen for and ingest transaction results from the connected digitalbits-core.

To enable ingestion, you must either pass `--ingest=true` on the command line or set the `INGEST`
environment variable to "true". Since version 1.0.0 you can start multiple ingesting machines in your cluster.

### Ingesting historical data and reingesting Ledgers

Reingesting older ledgers (due to a version upgrade) or ingesting ledgers closed by the network before Frontier was started is done through the `frontier db reingest range [START_LEDGER] [END_LEDGER]` command. This can be run as follows:

```
frontier1> frontier db reingest range 1 10000
frontier2> frontier db reingest range 10001 20000
frontier3> frontier db reingest range 20001 30000
# ... etc.
```

This allows reingestion to be split up and done in parallel by multiple Frontier processes.

### Managing storage for historical data

Over time, the recorded network history will grow unbounded, increasing storage used by the database. Frontier expands the data ingested from digitalbits-core and needs sufficient disk space. Unless you need to maintain a history archive you may configure Frontier to only retain a certain number of ledgers in the database. This is done using the `--history-retention-count` flag or the `HISTORY_RETENTION_COUNT` environment variable. Set the value to the number of recent ledgers you wish to keep around, and every hour the Frontier subsystem will reap expired data.  Alternatively, you may execute the command `frontier db reap` to force a collection.

### Surviving digitalbits-core downtime

Frontier tries to maintain a gap-free window into the history of the digitalbits-network.  This reduces the number of edge cases that Frontier-dependent software must deal with, aiming to make the integration process simpler.  To maintain a gap-free history, Frontier needs access to all of the metadata produced by digitalbits-core in the process of closing a ledger, and there are instances when this metadata can be lost.  Usually, this loss of metadata occurs because the digitalbits-core node went offline and performed a catchup operation when restarted.

To ensure that the metadata required by Frontier is maintained, you have several options: You may either set the `CATCHUP_COMPLETE` digitalbits-core configuration option to `true` or configure `CATCHUP_RECENT` to determine the amount of time your digitalbits-core can be offline without having to rebuild your Frontier database.

Unless your node is a full validator and archive publisher we _do not_ recommend using the `CATCHUP_COMPLETE` method, as this will force digitalbits-core to apply every transaction from the beginning of the ledger, which will take an ever-increasing amount of time. Instead, we recommend you set the `CATCHUP_RECENT` config value. To do this, determine how long of a downtime you would like to survive (expressed in seconds) and divide by ten.  This roughly equates to the number of ledgers that occur within your desired grace period (ledgers roughly close at a rate of one every ten seconds).  With this value set, digitalbits-core will replay transactions for ledgers that are recent enough, ensuring that the metadata needed by Frontier is present.

### Correcting gaps in historical data

In the section above, we mentioned that Frontier _tries_ to maintain a gap-free window.  Unfortunately, it cannot directly control the state of digitalbits-core and so gaps may form due to extended down time.  When a gap is encountered, Frontier will stop ingesting historical data and complain loudly in the log with error messages (log lines will include "ledger gap detected").  To resolve this situation, you must re-establish the expected state of the digitalbits-core database and purge historical data from Frontier's database.  We leave the details of this process up to the reader as it is dependent upon your operating needs and configuration, but we offer one potential solution:

We recommend you configure the HISTORY_RETENTION_COUNT in Frontier to a value less than or equal to the configured value for CATCHUP_RECENT in digitalbits-core.  Given this situation any downtime that would cause a ledger gap will require a downtime greater than the amount of historical data retained by Frontier.  To re-establish continuity:

1.  Stop Frontier.
2.  Run `frontier db reap` to clear the historical database.
3.  Clear the cursor for Frontier by running `digitalbits-core -c "dropcursor?id=FRONTIER"` (ensure capitilization is maintained).
4.  Clear ledger metadata from before the gap by running `digitalbits-core -c "maintenance?queue=true"`.
5.  Restart Frontier.

### Some endpoints are not available during state ingestion

Endpoints that display state information are not available during initial state ingestion and will return a `503 Service Unavailable`/`Still Ingesting` error.  An example is the `/paths` endpoint (built using offers). Such endpoints will become available after state ingestion is done (usually within a couple of minutes).

### State ingestion is taking a lot of time

State ingestion shouldn't take more than a couple of minutes on an AWS [`c5.xlarge` instance](https://aws.amazon.com/ec2/instance-types/c5/), or equivalent.

It's possible that the progress logs (see below) will not show anything new for a longer period of time or print a lot of progress entries every few seconds. This happens because of the way history archives are designed. The ingestion is still working but it's processing entries of type `DEADENTRY`. If there are a lot of them in the bucket, there are no _active_ entries to process. We plan to improve the progress logs to display actual percentage progress so it's easier to estimate ETA.

If you see that ingestion is not proceeding for a very long period of time:
1. Check the RAM usage on the machine. It's possible that the system ran out of RAM and is using swap memory, which is extremely slow.
2. If above is not the case, file a [new issue](https://github.com/xdbfoundation/go/issues/new/choose) in this repository.

### CPU usage goes high every few minutes

This is _by design_. Frontier runs a state verifier routine that compares state in local storage to history archives every 64 ledgers to ensure data changes are applied correctly. If data corruption is detected Frontier will block access to endpoints serving invalid data.

We recommend to keep this security feature turned on; however, if it's causing problems (due to CPU usage) this can be disabled with the `--ingest-disable-state-verification` CLI param or `INGEST_DISABLE_STATE_VERIFICATION` env variable.

### I see `Waiting for the next checkpoint...` messages

If you were running the new system in the past during experimental stage (`ENABLE_EXPERIMENTAL_INGESTION` flag) it's possible that the old and new systems are not in sync. In such case, the upgrade code will activate and will make sure the data is in sync. When this happens you may see `Waiting for the next checkpoint...` messages for up to 5 minutes.

## Reading the logs

In order to check the progress and the status of experimental ingestion you should check the logs. All logs connected to experimental ingestion are tagged with `service=ingest`.

It starts with informing you about state ingestion:

```
INFO[2019-08-29T13:04:13.473+02:00] Starting ingestion system from empty state...  pid=5965 service=ingest temp_set="*io.MemoryTempSet"
INFO[2019-08-29T13:04:15.263+02:00] Reading from History Archive Snapshot         ledger=25565887 pid=5965 service=ingest
```

During state ingestion, Frontier will log number of processed entries every 100,000 entries (there are currently around 1M entries in the public network):

```
INFO[2019-08-29T13:04:34.652+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=100000 pid=5965 service=ingest
INFO[2019-08-29T13:04:38.487+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=200000 pid=5965 service=ingest
INFO[2019-08-29T13:04:41.322+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=300000 pid=5965 service=ingest
INFO[2019-08-29T13:04:48.429+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=400000 pid=5965 service=ingest
INFO[2019-08-29T13:05:00.306+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=500000 pid=5965 service=ingest
```

When state ingestion is finished it will proceed to ledger ingestion starting from the next ledger after checkpoint ledger (25565887+1 in this example) to update the state using transaction meta:

```
INFO[2019-08-29T13:39:41.590+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=5300000 pid=5965 service=ingest
INFO[2019-08-29T13:39:44.518+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=5400000 pid=5965 service=ingest
INFO[2019-08-29T13:39:47.488+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=5500000 pid=5965 service=ingest
INFO[2019-08-29T13:40:00.670+02:00] Processed ledger                              ledger=25565887 pid=5965 service=ingest type=state_pipeline
INFO[2019-08-29T13:40:00.670+02:00] Finished processing History Archive Snapshot  duration=2145.337575904 ledger=25565887 numEntries=5529931 pid=5965 service=ingest shutdown=false
INFO[2019-08-29T13:40:00.693+02:00] Reading new ledger                            ledger=25565888 pid=5965 service=ingest
INFO[2019-08-29T13:40:00.694+02:00] Processing ledger                             ledger=25565888 pid=5965 service=ingest type=ledger_pipeline updating_database=true
INFO[2019-08-29T13:40:00.779+02:00] Processed ledger                              ledger=25565888 pid=5965 service=ingest type=ledger_pipeline
INFO[2019-08-29T13:40:00.779+02:00] Finished processing ledger                    duration=0.086024492 ledger=25565888 pid=5965 service=ingest shutdown=false transactions=14
INFO[2019-08-29T13:40:00.815+02:00] Reading new ledger                            ledger=25565889 pid=5965 service=ingest
INFO[2019-08-29T13:40:00.816+02:00] Processing ledger                             ledger=25565889 pid=5965 service=ingest type=ledger_pipeline updating_database=true
INFO[2019-08-29T13:40:00.881+02:00] Processed ledger                              ledger=25565889 pid=5965 service=ingest type=ledger_pipeline
INFO[2019-08-29T13:40:00.881+02:00] Finished processing ledger                    duration=0.06619956 ledger=25565889 pid=5965 service=ingest shutdown=false transactions=29
INFO[2019-08-29T13:40:00.901+02:00] Reading new ledger                            ledger=25565890 pid=5965 service=ingest
INFO[2019-08-29T13:40:00.902+02:00] Processing ledger                             ledger=25565890 pid=5965 service=ingest type=ledger_pipeline updating_database=true
INFO[2019-08-29T13:40:00.972+02:00] Processed ledger                              ledger=25565890 pid=5965 service=ingest type=ledger_pipeline
INFO[2019-08-29T13:40:00.972+02:00] Finished processing ledger                    duration=0.071039012 ledger=25565890 pid=5965 service=ingest shutdown=false transactions=20
```


## Managing Stale Historical Data

Frontier ingests ledger data from a connected instance of digitalbits-core.  In the event that digitalbits-core stops running (or if Frontier stops ingesting data for any other reason), the view provided by Frontier will start to lag behind reality.  For simpler applications, this may be fine, but in many cases this lag is unacceptable and the application should not continue operating until the lag is resolved.

To help applications that cannot tolerate lag, Frontier provides a configurable "staleness" threshold.  Given that enough lag has accumulated to surpass this threshold (expressed in number of ledgers), Frontier will only respond with an error: [`stale_history`](./reference/errors/stale-history.md).  To configure this option, use either the `--history-stale-threshold` command line flag or the `HISTORY_STALE_THRESHOLD` environment variable.  NOTE:  non-historical requests (such as submitting transactions or finding payment paths) will not error out when the staleness threshold is surpassed.

## Monitoring

To ensure that your instance of Frontier is performing correctly we encourage you to monitor it, and provide both logs and metrics to do so.

Frontier will output logs to standard out.  Information about what requests are coming in will be reported, but more importantly, warnings or errors will also be emitted by default.  A correctly running Frontier instance will not output any warning or error log entries.

Metrics are collected while a Frontier process is running and they are exposed at the `/metrics` path through the Frontier admin port. You need to configure this via `--admin-port` or `ADMIN_PORT`, since it’s disabled by default. 

Below we present a few standard log entries with associated fields. You can use them to build metrics and alerts. We present below some examples. Please note that this represents Frontier app metrics only. You should also monitor your hardware metrics like CPU or RAM Utilization.

### Starting HTTP request

| Key              | Value                                                                                          |
|------------------|------------------------------------------------------------------------------------------------|
| **`msg`**        | **`Starting request`**                                                                         |
| `client_name`    | Value of `X-Client-Name` HTTP header representing client name                                  |
| `client_version` | Value of `X-Client-Version` HTTP header representing client version                            |
| `app_name`       | Value of `X-App-Name` HTTP header representing app name                                        |
| `app_version`    | Value of `X-App-Version` HTTP header representing app version                                  |
| `forwarded_ip`   | First value of `X-Forwarded-For` header                                                        |
| `host`           | Value of `Host` header                                                                         |
| `ip`             | IP of a client sending HTTP request                                                            |
| `ip_port`        | IP and port of a client sending HTTP request                                                   |
| `method`         | HTTP method (`GET`, `POST`, ...)                                                               |
| `path`           | Full request path, including query string (ex. `/transactions?order=desc`)                     |
| `streaming`      | Boolean, `true` if request is a streaming request                                              |
| `referer`        | Value of `Referer` header                                                                      |
| `req`            | Random value that uniquely identifies a request, attached to all logs within this HTTP request |

### Finished HTTP request

| Key              | Value                                                                                          |
|------------------|------------------------------------------------------------------------------------------------|
| **`msg`**        | **`Finished request`**                                                                         |
| `bytes`          | Number of response bytes sent                                                                  |
| `client_name`    | Value of `X-Client-Name` HTTP header representing client name                                  |
| `client_version` | Value of `X-Client-Version` HTTP header representing client version                            |
| `app_name`       | Value of `X-App-Name` HTTP header representing app name                                        |
| `app_version`    | Value of `X-App-Version` HTTP header representing app version                                  |
| `duration`       | Duration of request in seconds                                                                 |
| `forwarded_ip`   | First value of `X-Forwarded-For` header                                                        |
| `host`           | Value of `Host` header                                                                         |
| `ip`             | IP of a client sending HTTP request                                                            |
| `ip_port`        | IP and port of a client sending HTTP request                                                   |
| `method`         | HTTP method (`GET`, `POST`, ...)                                                               |
| `path`           | Full request path, including query string (ex. `/transactions?order=desc`)                     |
| `route`          | Route pattern without query string (ex. `/accounts/{id}`)                                      |
| `status`         | HTTP status code (ex. `200`)                                                                   |
| `streaming`      | Boolean, `true` if request is a streaming request                                              |
| `referer`        | Value of `Referer` header                                                                      |
| `req`            | Random value that uniquely identifies a request, attached to all logs within this HTTP request |

### Metrics

Using the entries above you can build metrics that will help understand performance of a given Frontier node, some examples below:

- Number of requests per minute.
- Number of requests per route (the most popular routes).
- Average response time per route.
- Maximum response time for non-streaming requests.
- Number of streaming vs. non-streaming requests.
- Number of rate-limited requests.
- List of rate-limited IPs.
- Unique IPs.
- The most popular SDKs/apps sending requests to a given Frontier node.
- Average ingestion time of a ledger.
- Average ingestion time of a transaction.

### Alerts

Below we present example alerts with potential cause and solution. Feel free to add more alerts using your metrics.

Alert | Cause | Solution
-|-|-
Spike in number of requests | Potential DoS attack | Lower rate-limiting threshold
Large number of rate-limited requests | Rate-limiting threshold too low | Increase rate-limiting threshold
Ingestion is slow | Frontier server spec too low | Increase hardware spec
Spike in average response time of a single route | Possible bug in a code responsible for rendering a route | Report an issue in Frontier repository.