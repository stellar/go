# Migration 
In this section, we'll discuss migrating existing systems running the pre-2.0 versions of Horizon to the new 2.x world.

**Environment assumptions**:

  - We assume that the **PostgreSQL server** lives at the `db.local` hostname, and has a `horizon` database accessible by the `postgres:secret` username-password combo.

  - We assume your machine has **enough extra RAM** to hold Captive Core's in-memory database (~3GiB), which is a larger memory requirement than a traditional Core setup (which would have an on-disk database).

  - The examples here refer to the **testnet for safety**; replace the appropriate references with the pubnet equivalents when you're ready.

  - In some places, bleeding-edge versions of packages can be built from scratch. We assume a sane **[Golang](https://golang.org/doc/install) development environment** for this.

To start off simply, we assume a **single-machine Ubuntu setup** running both Horizon and Core with a single local PostgreSQL server. This assumption is loosened in a [later section](#multi-machine-setup).


## Installing
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


## Upgrading
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

### Configure Horizon
First, add the following lines to the Horizon configuration to enable a Captive Core subprocess:

```bash
echo "STELLAR_CORE_BINARY_PATH=$(which stellar-core)
CAPTIVE_CORE_CONFIG_APPEND_PATH=/etc/default/stellar-captive-core.toml" | sudo tee -a /etc/default/stellar-horizon
```

(Note that setting `ENABLE_CAPTIVE_CORE_INGESTION=true` is not necessary in 2.x because it's the new default.)


**Note**: Depending on the version you're migrating from, you may need to include an additional step here: manual reingestion. This can still be accomplished with Captive Core; see [below](#reingestion).

### Restarting Services
Now, we can stop Core (which hopefully doesn't need an explanation) and restart Horizon:

```bash
stellar-horizon-cmd serve
```

The logs should show Captive Core running successfully as a subprocess, and eventually Horizon will be running as usual, except with Captive Core rapidly generating transaction metadata in-memory!


## Multi-Machine Setup
If you plan on running Horizon and Captive Core on separate machines, you'll need to change only a few things. Namely, rather than configuring the `STELLAR_CORE_BINARY` variable, you'll need to point Horizon at the Remote Captive Core instance via `REMOTE_CAPTIVE_CORE_URL` (for the wrapper API) and `STELLAR_CORE_URL` (for the raw Core API).

In this section, we'll work through a hypothetical architecture with two Horizon instances (only one of which does ingestion) and a single Captive Core instance.

### Remote Captive Core
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

### Ingestion Instance
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

Then just run it as usual:

```
stellar-horizon-cmd serve
```

### Serving Instance
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
stellar-horizon-cmd serve
```

At this point, you should be able to hit port 8000 on the above instance and watch the `ingest_latest_ledger` value grow.


# Reingestion
If you need to manually reingest some ledgers (for example, you want history for some ledgers that closed before your asset got issued), you can still do this with Captive Core.

For example, suppose we've ingested from ledger 811520, but would like another 1000 ledgers before it to be ingested as well. Nothing really changes from the execution perspective relative to the "old" way (given the configuration updates [from before](#configure-horizon) are done):

```bash
stellar-horizon-cmd db reingest range 810520 811520
```

The biggest change is simply how much faster this gets done! For example, a [full reingestion](#using-captive-core-to-reingest-the-full-public-network-history) of the entire network only takes ~1.5 days (as opposed to weeks previously) on an [m5.8xlarge](https://aws.amazon.com/ec2/pricing/on-demand/) instance. :fire:
