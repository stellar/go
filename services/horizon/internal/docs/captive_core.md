---
title: Using Captive Stellar-Core in Horizon
replacement: https://developers.stellar.org/docs/horizon-captive-core/
---

## Using Captive Stellar-Core in Horizon

### Table of Contents

  * [Introduction](#introduction)
  * [Configuration](#configuration)
  * [Requirements](#requirements)
  * [How It Works](#how-it-works)
  * [Known Issues](#known-issues)
  * [Migration](#migration)
  * [Reingestion](#reingestion)

### Introduction

Starting with version v1.6.0, Horizon allows using Stellar-Core in captive mode. Core's captive mode is enabled by default since Horizon 2.0.

When Captive Core mode is disabled, this is how Horizon interacts with Core:

![Horizon without Captive Core](HorizonWithoutCaptiveCore.png)

Captive Core mode relaxes Horizon's operational requirements. It allows running Horizon without a fully fledged Core instance and, most importantly, without a Core database. 

Captive Core is a specialized narrowed-down Stellar-Core instance with the sole aim of emitting transaction metadata (often abbreviated in documentation as "txmeta") to Horizon. Captive Core does all the work in-memory, without a database; for this reason, it has higher memory requirements (which will be covered in a section below) but is much more performant.

The Captive Core instance can be started as:

A. a subprocess of Horizon, streaming ledger data over a filesystem pipe. This is the default:

![Horizon with Captive Core](HorizonWithCaptiveCore.png)
    
B. an experimental remote captive core server, to which Horizon connects through HTTP (see Horizon's `--remote-captive-core-url` flag).
    For more information on installing the remote Captive Core server, please take a look at [captivecore](https://github.com/stellar/go/tree/master/exp/services/captivecore):

![Horizon with Remote Captive Core](HorizonWithRemoteCaptiveCore.png)

Captive Core completely eliminates all Horizon issues connected Stellar-Core's database, but it requires extra time to initialize the Stellar-Core subprocess.

Captive Stellar-Core can be used in both reingestion (`horizon db reingest range`) and normal Horizon operation (`horizon` or `horizon serve`). In fact, using Captive Core to reingest historical data is considerably faster than without it (i.e. filling in Core's database with `stellar-core catchup` followed by `horizon db reingest range`).

### Configuration

To enable captive mode you will need to initialize some configuration variables:

* (required) `ENABLE_CAPTIVE_CORE_INGESTION=true` (enabled by default since Horizon 2.0).
* (required) If you run Captive Core ...

  A. ... as a subprocess:
     * `STELLAR_CORE_BINARY_PATH` - defines a path to the `stellar-core` binary,
     * (not required when running `horizon db reingest range`) `CAPTIVE_CORE_CONFIG_APPEND_PATH` - defines a path to a file to append to the Stellar Core configuration file used by captive core. It must, at least, include enough details to define a quorum set. Refer [below](#configure-captive-core) for an example stub connecting to the Stellar testnet SDF servers.
  
  B. ... using the experimental Captive Core server:
     * `REMOTE_CAPTIVE_CORE_URL` - URL to access the remote captive core server
  
* (optional) `CAPTIVE_CORE_HTTP_PORT` - HTTP port for Captive Core to listen on (0 disables the HTTP server)


### Requirements

* An additional 3GB of RAM,
* Horizon v1.6.0,
* Stellar-Core v13.2.0.

### How It Works

When using Captive Stellar-Core, Horizon runs the `stellar-core` binary as a subprocess. Then both processes communicate over filesystem pipe: Stellar-Core sends `xdr.LedgerCloseMeta` structs with information about each ledger and Horizon reads it.

The behaviour is slightly different when reingesting old ledgers and when reading recently closed ledgers.

When reingesting, Stellar-Core is started in a special `catchup` mode that simply replays the requested range of ledgers. This mode requires an additional 3GB of RAM because all ledger entries are stored in memory, making it extremely fast. This mode only depends on the history archives, so a Stellar-Core configuration file is not required.

When reading recently closed ledgers, Stellar-Core is started with a normal `run` command. This requires a persistent database, so extra RAM is not needed, but it makes the initial stage of applying buckets slower than when reingesting. In this case a Stellar-Core configuration file **is** required to configure a quorum set so that it can connect to the Stellar network.

### Known Issues

As discussed earlier, Captive Core provides much better decoupling for Horizon at the expense of persistence. You should be aware of the following consequences:

* Captive Stellar-Core requires a couple of minutes to complete the apply buckets stage every time Horizon in started.
* If the Horizon process terminates, Stellar-Core is also terminated.
* Requires extra RAM.

To mitigate this we recommend running multiple ingesting Horizon servers in a single cluster. This allows other ingesting instances to maintain service without interruptions if a Captive Stellar-Core is restarted.

### Migration

In this section, we'll discuss migrating existing systems running the pre-2.0 versions of Horizon to the new 2.x world.

**Environment assumptions**:

  - We assume that the **PostgreSQL server** lives at the `db.local` hostname, and has a `horizon` database accessible by the `postgres:secret` username-password combo.

  - We assume your machine has **enough extra RAM** to hold Captive Core's in-memory database (~3GB), which is a larger memory requirement than a traditional Core setup (which would have an on-disk database).

  - The examples here refer to the **testnet for safety**; replace the appropriate references with the pubnet equivalents when you're ready.

  - In some places, bleeding-edge versions of packages can be built from scratch. We assume a sane **[Golang](https://golang.org/doc/install) development environment** for this.

To start off simply, we assume a **single-machine Ubuntu setup** running both Horizon and Core with a single local PostgreSQL server. This assumption is loosened in a [later section](#multi-machine-setup).


#### Installing
The process for upgrading both Stellar Core and Horizon are covered [here](https://github.com/stellar/packages/blob/master/docs/upgrading.md#upgrading). The only difference is that since we're migrating to a beta Horizon release, we need to first add the unstable repository; this is described [here](https://github.com/stellar/packages/blob/master/docs/adding-the-sdf-stable-repository-to-your-system.md#adding-the-bleeding-edge-unstable-repository).

Then, you can install the Captive Core packages:

```bash
sudo apt install stellar-captive-core
```

And you're ready to upgrade.

**Note**: Until the v2.0-beta binaries are released, you'll need to build Horizon from the `release-horizon-v2.0.0-beta` branch. That's pretty simple, given a valid Go environment:

```bash
git clone https://github.com/stellar/go monorepo && cd monorepo
git checkout release-horizon-v2.0.0-beta
go install -v ./services/horizon
sudo cp $(go env GOPATH)/bin/horizon $(which stellar-horizon)
```


#### Upgrading
At this point, all that is left to do is to:

 - create a Captive Core configuration stub
 - modify the Horizon configuration to enable Captive Core (we will assume it lives in `/etc/default/stellar-horizon`, the default)
 - stop the existing Stellar Core instance
 - restart Horizon

### Configure Captive Core
Captive Core runs with a trimmed down configuration "stub": at minimum, it must contain enough info to set up a quorum (see [above](#configuration)). **Your old configuration cannot be used directly**: Horizon needs special settings for Captive Core. Otherwise, running Horizon will fail with the following error, or errors like it:

    default: Config from /tmp/captive-stellar-core-38cff455ad3469ec/stellar-core.conf
    default: Got an exception: Failed to parse '/tmp/captive-stellar-core-38cff455ad3469ec/stellar-core.conf' :Key HTTP_PORT already present at line 10 [CommandLine.cpp:1064]


For example, if relying exclusively on SDF's validators:

```toml
[[HOME_DOMAINS]]
HOME_DOMAIN="testnet.stellar.org"
QUALITY="HIGH"

[[VALIDATORS]]
NAME="sdf_testnet_1"
HOME_DOMAIN="testnet.stellar.org"
PUBLIC_KEY="GDKXE2OZMJIPOSLNA6N6F2BVCI3O777I2OOC4BV7VOYUEHYX7RTRYA7Y"
ADDRESS="core-testnet1.stellar.org"
HISTORY="curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_001/{0} -o {1}"

[[VALIDATORS]]
NAME="sdf_testnet_2"
HOME_DOMAIN="testnet.stellar.org"
PUBLIC_KEY="GCUCJTIYXSOXKBSNFGNFWW5MUQ54HKRPGJUTQFJ5RQXZXNOLNXYDHRAP"
ADDRESS="core-testnet2.stellar.org"
HISTORY="curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_002/{0} -o {1}"

[[VALIDATORS]]
NAME="sdf_testnet_3"
HOME_DOMAIN="testnet.stellar.org"
PUBLIC_KEY="GC2V2EFSXN6SQTWVYA5EPJPBWWIMSD2XQNKUOHGEKB535AQE2I6IXV2Z"
ADDRESS="core-testnet3.stellar.org"
HISTORY="curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_003/{0} -o {1}"
```

(We'll assume this stub lives at `/etc/default/stellar-captive-core.toml`.) The rest of the configuration will be generated automagically at runtime.

#### Configure Horizon
First, add the following lines to the Horizon configuration to enable a Captive Core subprocess:

```bash
echo "STELLAR_CORE_BINARY_PATH=$(which stellar-core)
CAPTIVE_CORE_CONFIG_APPEND_PATH=/etc/default/stellar-captive-core.toml" | sudo tee -a /etc/default/stellar-horizon
```

(Note that setting `ENABLE_CAPTIVE_CORE_INGESTION=true` is not necessary in 2.x because it's the new default.)


**Note**: Depending on the version you're migrating from, you may need to include an additional step here: manual reingestion. This can still be accomplished with Captive Core; see [below](#reingestion).

#### Restarting Services
Now, we can stop Core and restart Horizon:

```bash
sudo systemctl stop stellar-core
sudo systemctl restart stellar-horizon
```

The logs should show Captive Core running successfully as a subprocess, and eventually Horizon will be running as usual, except with Captive Core rapidly generating transaction metadata in-memory!


#### Multi-Machine Setup
If you plan on running Horizon and Captive Core on separate machines, you'll need to change only a few things. Namely, rather than configuring the `STELLAR_CORE_BINARY` variable, you'll need to point Horizon at the Remote Captive Core instance via `REMOTE_CAPTIVE_CORE_URL` (for the wrapper API) and `STELLAR_CORE_URL` (for the raw Core API).

In this section, we'll work through a hypothetical architecture with two Horizon instances (only one of which does ingestion) and a single Captive Core instance.

#### Remote Captive Core
First, we need to start running the Captive Core server.

The latest released (but experimental) version of the Captive Core API can be installed from the [unstable repo](https://github.com/stellar/packages/blob/master/docs/adding-the-sdf-stable-repository-to-your-system.md#adding-the-bleeding-edge-unstable-repository):

```bash
sudo apt install stellar-captive-core stellar-captive-core-api
```

Alternatively, you can install the bleeding edge [from source](https://github.com/stellar/go/exp/services/captivecore):

```bash
git clone https://github.com/stellar/go monorepo && cd monorepo
git checkout release-horizon-v2.0.0-beta
go install -v ./exp/services/captivecore
sudo cp $(go env GOPATH)/bin/captivecore /usr/bin/stellar-captive-core-api
```

Now, let's configure the Captive Core environment:

```bash
export NETWORK_PASSPHRASE='Test SDF Network ; September 2015'
export HISTORY_ARCHIVE_URLS='https://history.stellar.org/prd/core-testnet/core_testnet_001'
export DATABASE_URL='postgres://postgres:secret@db.local:5432/horizon?sslmode=disable'
export CAPTIVE_CORE_CONFIG_APPEND_PATH=/etc/default/stellar-captive-core.toml
export STELLAR_CORE_BINARY_PATH=$(which stellar-core)
```

(There's no `-cmd` wrapper à la Horizon/Core for this binary yet; you can run these commands directly or `source` them into your shell from a script.) The parameters should all be familiar from earlier sections.

Finally, let's run the Captive Core instance:

```bash
stellar-captive-core-api
```

This will start serving *two* endpoints: a Captive Core wrapper API on port 8000 (by default), which serves up processed ledgers and can be queried by Horizon, and the underlying Core API on port 11626 (by default). See the `--help` for how to configure the ports.

#### Ingestion Instance
Returning to the Horizon instance that will be doing ingestion, we just need to supply the appropriate URLs and ports. 

Suppose the above server can be resolved on the `captivecore.local` hostname; then, we need to configure Horizon accordingly:

```bash
echo "DATABASE_URL='postgres://postgres@db.local:5432/horizon?sslmode=disable'
HISTORY_ARCHIVE_URLS='https://history.stellar.org/prd/core-testnet/core_testnet_001'
NETWORK_PASSPHRASE='Test SDF Network ; September 2015'
INGEST=true
STELLAR_CORE_URL='http://captivecore.local:11626'
REMOTE_CAPTIVE_CORE_URL='http://captivecore.local:8000'
" | sudo tee /etc/default/stellar-horizon
```

Then just run it as usual, directly from the command-line or as a service:

```
stellar-horizon-cmd serve
# or
sudo systemctl restart stellar-horizon
```

#### Serving Instance
This configuration is almost identical, except we flip the ingestion parameters:

```bash
echo "DATABASE_URL='postgres://postgres@db.local:5432/horizon?sslmode=disable'
HISTORY_ARCHIVE_URLS='https://history.stellar.org/prd/core-testnet/core_testnet_001'
NETWORK_PASSPHRASE='Test SDF Network ; September 2015'
INGEST=false
ENABLE_CAPTIVE_CORE_INGESTION=false
STELLAR_CORE_URL='http://captivecore.local:11626'
REMOTE_CAPTIVE_CORE_URL='http://captivecore.local:8000'
" | sudo tee /etc/default/stellar-horizon
sudo systemctl restart stellar-horizon
```

At this point, you should be able to hit port 8000 on the above instance and watch the `ingest_latest_ledger` value grow.


### Reingestion
If you need to manually reingest some ledgers (for example, you want history for some ledgers that closed before your asset got issued), you can still do this with Captive Core.

For example, suppose we've ingested from ledger 811520, but would like another 1000 ledgers before it to be ingested as well. Nothing really changes from the execution perspective relative to the "old" way (given the configuration updates [from before](#configure-horizon) are done):

```bash
stellar-horizon-cmd db reingest range 810520 811520
```

The biggest change is simply how much faster this gets done! For example, a [full reingestion](#using-captive-core-to-reingest-the-full-public-network-history) of the entire network only takes ~1.5 days (as opposed to weeks previously) on an [m5.8xlarge](https://aws.amazon.com/ec2/pricing/on-demand/) instance. :fire:


#### Using Captive Core to reingest the full public network history

In some cases it can be convenient to (re)ingest the full Stellar Public
Network history into Horizon (e.g. when running Horizon for the first time).

This process used to take weeks.
However, using multiple Captive Core workers on a high performance
environment (powerful machine on which to run Horizon + powerful Database)
makes this possible in ~1.5 days.


The following instructions assume the reingestion is done on AWS.
However, they should be applicable to any other environment with equivalent
capacity. In the same way, the instructions can be adapted to reingest only
specify parts of the history.

##### Prerequisites

1. An `m5.8xlarge` (32 cores, 64GB of RAM) EC2 instance with at least 200 GB 
   of disk capacity from which to run Horizon.
   This is needed to fit 24 Horizon parallel workers (each with its own
   Captive Core instance). Each Core instance can take up to 3GB of RAM and a
   full core (more on why 24 workers below). If the number of workers is
   increased, you may need a larger machine.
   
2. Horizon version 1.6.0 or newer installed in the machine from (1).

3. [Core](https://github.com/stellar/stellar-core) version 13.0 or newer installed
   in the machine from (1).

4. A Horizon database, where to reingest the History. Preferably, the
   database should be at least an RDS `r4.8xlarge` instance or better (to take
   full advantage of its IOPS write capacity) and
   should be empty, to minimize storage (Postgres accumulates data during
   usage, which is only deleted when `VACUUM`ed). When using an RDS instance
   with General Purpose SSD storage, the reingestion throughput of the DB
   (namely Write IOPS) is determined by the storage size (3 IOPS per GB).
   With 5TB you get 15K IOPS, which can be saturated with 24 Horizon
   workers. As the DB storage grows,
   the IO capacity will grow along with it. The number of workers (and the
   size of the instance created in (1), should be increased accordingly if
   we want to take advantage of it. To make sure we are minimizing the
   reingestion time, we should look at the RDS _Write IOPS_ CloudWatch graph.
   The graph should ideally always be close to the theoretical limit of
   the DB (3000 IOPS per TB of storage).


##### Reingestion

Once the prerequisites are satisfied, we can spawn two Horizon reingestion
processes in parallel:

 1. One for the first 17 million ledgers (which are almost empty).
 2. Another one for the rest of the history.

This is due to first 17 million ledgers being almost empty whilst the rest
are much more packed. Having a single Horizon instance with enough workers to
saturate the IO capacity of the machine for the first 17 million would kill the
machine when reingesting the rest (during which there is a higher CPU and
memory consumption per woker).

64 workers for (1) and 24 workers for (2) saturates the IO capacity of an RDS
instance with 5TB of General Purpose SSD storage. Again, as the DB storage
grows, a larger number of workers should be considered.

In order to run the reingestion, first set the following environment variables in the configuration:

```bash
export DATABASE_URL=postgres://postgres:secret@db.local:5432/horizon
export APPLY_MIGRATIONS=true
export HISTORY_ARCHIVE_URLS=https://s3-eu-west-1.amazonaws.com/history.stellar.org/prd/core-live/core_live_001
export NETWORK_PASSPHRASE="Public Global Stellar Network ; September 2015"
export STELLAR_CORE_BINARY_PATH=$(which stellar-core)
export ENABLE_CAPTIVE_CORE_INGESTION=true
# Number of ledgers per job sent to the workers
# The larger the job, the better performance from Captive Core's perspective, but, you want to choose a job size which maximizes the time all workers are busy. 
export PARALLEL_JOB_SIZE=100000
# Retries per job
export RETRIES=10
export RETRY_BACKOFF_SECONDS=20
```

(Naturally, you can also edit the configuration file at `/etc/default/stellar-horizon` directly.) If Horizon was previously running, ensure it is stopped. Then, run the following commands in parallel:

1. `stellar-horizon db reingest range --parallel-workers=64 1 16999999`
2. `stellar-horizon db reingest range --parallel-workers=24 17000000 <latest_ledger>`

When saturating an RDS instance with 15K IOPS capacity:

(1) should take a few hours to complete.

(2) should take about 1.5 days to complete.

Although there is a retry mechanism, reingestion may fail half-way. Horizon will print the recommended range to use in order to restart it. 


##### Monitoring reingestion process

This script should help monitor the reingestion process by printing the ledger subranges being reingested:

```bash
#!/bin/bash
echo "Current ledger ranges being reingested:"
echo
I=1
for S in $(ps aux | grep stellar-core | grep catchup | awk '{print $15}' | sort -n); do
    printf '%15s' $S
    if [ $(( I % 5 )) = 0 ]; then
      echo
    fi
    I=$(( I + 1))
done
```
 
Ideally we would be using Prometheus metrics for this, but they haven't been
implemented yet.

Here is an example run:

```
Current ledger ranges being reingested:
    99968/99968   199936/99968   299904/99968   399872/99968   499840/99968
   599808/99968   699776/99968   799744/99968   899712/99968   999680/99968
  1099648/99968  1199616/99968  1299584/99968  1399552/99968  1499520/99968
  1599488/99968  1699456/99968  1799424/99968  1899392/99968  1999360/99968
  2099328/99968  2199296/99968  2299264/99968  2399232/99968  2499200/99968
  2599168/99968  2699136/99968  2799104/99968  2899072/99968  2999040/99968
  3099008/99968  3198976/99968  3298944/99968  3398912/99968  3498880/99968
  3598848/99968  3698816/99968  3798784/99968  3898752/99968  3998720/99968
  4098688/99968  4198656/99968  4298624/99968  4398592/99968  4498560/99968
  4598528/99968  4698496/99968  4798464/99968  4898432/99968  4998400/99968
  5098368/99968  5198336/99968  5298304/99968  5398272/99968  5498240/99968
  5598208/99968  5698176/99968  5798144/99968  5898112/99968  5998080/99968
  6098048/99968  6198016/99968  6297984/99968  6397952/99968 17099967/99968
 17199935/99968 17299903/99968 17399871/99968 17499839/99968 17599807/99968
 17699775/99968 17799743/99968 17899711/99968 17999679/99968 18099647/99968
 18199615/99968 18299583/99968 18399551/99968 18499519/99968 18599487/99968
 18699455/99968 18799423/99968 18899391/99968 18999359/99968 19099327/99968
 19199295/99968 19299263/99968 19399231/99968
```
