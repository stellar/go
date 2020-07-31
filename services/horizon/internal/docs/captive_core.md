---
title: Using Captive Stellar-Core in Horizon
replacement: https://developers.stellar.org/docs/horizon-captive-core/
---
## Using Captive Stellar-Core in Horizon

Starting with version v1.6.0, Horizon contains an experimental feature to use Stellar-Core in the captive mode. In this mode Stellar-Core is started as a subprocess of Horizon and streams ledger data over a filesystem pipe. It completely eliminates all issues connected to ledgers missing in Stellar-Core database but requires an extra time for init stage.

Captive Stellar-Core can be used in both reingestion and normal Horizon operations. To learn more how captive mode can make reingestion faster check TBA.

### Configuration

To enable captive mode three feature config variables are required:
* `ENABLE_CAPTIVE_CORE_INGESTION=true`,
* `STELLAR_CORE_BINARY_PATH` - defines a path to `stellar-core` binary,
* `STELLAR_CORE_CONFIG_PATH` - defines a path to `stellar-core.cfg` file (not required when reingesting).

### Requirements

* Additional 3GB of RAM,
* Horizon v1.6.0,
* Stellar-Core v13.2.0.

### How it works?

When using Captive Stellar-Core, Horizon runs `stellar-core` binary as a subprocess. Then both processes communicate over filesystem pipe: Stellar-Core sends `xdr.LedgerCloseMeta` structs with information about each ledger and Horizon reads it.

The behaviour is slightly different when reingesting old ledgers and when reading recently closed ledgers.

When reingesting, Stellar-Core is started in `catchup` mode that simply replays the requested range of ledgers. This mode requires additional 3GB of RAM because all ledger entries are stored in memory - making it extremely fast. This mode is using history archives only so `stellar-core.cfg` file is not required.

When reading recently closed ledgers, Stellar-Core is started with a normal `run` command. It requires a persistent database thus extra RAM is not needed but it makes the initial state of applying buckets slower than when reingesting. Also, `stellar-core.cfg` is required because Stellar-Core needs to connect to Stellar network so it needs a quorum set configuration.

### Known Issues

As mentioned in the first section, Captive Stellar-Core provides better decoupling however may introduce some issues:

* Requires a couple minutes for apply buckets stage every time Horizon in started.
* If Horizon process is terminated, Stellar-Core is also terminated.
* Requires extra RAM.

To mitigate this we recommend running multiple ingesting Horizon servers in a single cluster.