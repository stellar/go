# Migration 
In this section, we'll discuss migrating existing systems running the [latest](https://github.com/stellar/go/releases/latest) stable version of Horizon ([1.13](https://github.com/stellar/go/releases/tag/horizon-v1.13.0) as of this writing) to the new 2.0 beta. 

**Environment assumptions**:

  - We assume that the **PostgreSQL server** lives at the `db.local` hostname, and has a `horizon` database accessible by the `postgres:secret` username-password combo.

  - We assume your machine has **enough memory** to hold Captive Core's in-memory database (~3GiB), which is a larger memory requirement than a traditional Core setup (which would have an on-disk database).

  - The examples here refer to the testnet for safety; replace the appropriate references with the pubnet equivalents when you're ready.

To start off simply, we assume a **single-machine Ubuntu setup** running both Horizon and Core with a single local PostgreSQL server. Then, this assumption is loosened in a [later section](#multi-machine-setup).


## Installing
The process for upgrading both Stellar Core and Horizon are covered [here](https://github.com/stellar/packages/blob/master/docs/upgrading.md#upgrading); the only difference is that since we're migrating to Horizon v2.0-beta, we need to first add the unstable repository. This is described [here](https://github.com/stellar/packages/blob/master/docs/adding-the-sdf-stable-repository-to-your-system.md#adding-the-bleeding-edge-unstable-repository), but in brief, you just need to add the URL:

```bash
echo "deb https://apt.stellar.org xenial unstable" | sudo tee -a /etc/apt/sources.list.d/SDF-unstable.list
```

Then, you can install the Captive Core packages:

```bash
sudo apt install stellar-captive-core
```

And you're ready to upgrade.

**Note**: Until v2.0-beta is tagged & released, you'll need to build Horizon from master for this to actually render the latest binaries. That's done pretty simply, given a valid Go environment:

```bash
git clone https://github.com/stellar/go && cd go
go install -v ./services/horizon
sudo cp $GOPATH/bin/horizon $(which stellar-horizon)
```


## Upgrading
At this point, all that is left to do is to:

 - modify the Horizon configuration to enable Captive Core (we will assume it lives in `/etc/default/stellar-horizon`, the default)
 - create a Captive Core configuration stub
 - stop the existing Stellar Core instance
 - restart Horizon

### Configure Captive Core
Captive Core runs with a trimmed down configuration "stub": at minimum, it must contain enough info to set up a quorum (see [above](#todo-fons-section-link)). For example, if relying exclusively on SDF's validators:

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

**Note:** Using your existing Stellar Core configuration will not work (**why not???**). Running Horizon will fail with the following error, or errors like it:

    default: Config from /tmp/captive-stellar-core-38cff455ad3469ec/stellar-core.conf
    default: Got an exception: Failed to parse '/tmp/captive-stellar-core-38cff455ad3469ec/stellar-core.conf' :Key HTTP_PORT already present at line 10 [CommandLine.cpp:1064]

### Configure Horizon
First, add the following lines to the Horizon configuration to enable a Captive Core subprocess:

```bash
echo "STELLAR_CORE_BINARY_PATH=$(which stellar-core)
CAPTIVE_CORE_CONFIG_APPEND_PATH=/etc/default/stellar-captive-core.toml" | sudo tee -a /etc/default/stellar-horizon
```

(Note that setting `ENABLE_CAPTIVE_CORE_INGESTION=1` is not necessary as of Horizon 2.0 because it's the default. TODO: This isn't true on master, so gotta make sure.)


**Note**: There may be an additional step necessary here if you aren't coming from v1.13, depending on the changelog: manual reingestion. You can still accomplish this with Captive Core, see [below](#reingestion).


### Restarting Services
Now, we can stop Core (which hopefully doesn't need an explanation) and restart Horizon:

```bash
stellar-horizon-cmd serve
```

The logs should show Captive Core running successfully as a subprocess, and eventually Horizon will be running as usual, except with Captive Core rapidly generating transaction metadata in-memory!


## Multi-Machine Setup
If you plan on running Horizon and Captive Core on separate machines, you'll need to change only a few things. Namely, rather than configuring the `STELLAR_CORE_BINARY` variable, you'll need to point Horizon at the Remote Captive Core instance via `REMOTE_CAPTIVE_CORE_URL`.

In this section, we'll work through a hypothetical architecture with two Horizon instances, one of which is an ingestion instance, and a single Captive Core instance.

### Remote Captive Core
First, we need to start running the Captive Core server.

This must be installed from source, as it's not a published package yet (TODO: right?). These instructions presume a "sane" Golang environment (namely, one with `$GOPATH` defined):

```bash
git clone https://github.com/stellar/go && cd go
go install -v ./exp/services/captivecore
sudo cp $GOPATH/bin/captivecore /usr/bin/stellar-captive-core
```

Now, let's run a Captive Core instance:

```bash
stellar-captive-core \
  --network-passphrase='Test SDF Network ; September 2015' \
  --history-archive-urls='https://history.stellar.org/prd/core-testnet/core_testnet_001' \
  --db-url='postgres://postgres:secret@db.local:5432/horizon?sslmode=disable' \
  --port=8080 \
  --stellar-core-binary-path=$(which stellar-core) \
  --captive-core-config-append-path=/etc/default/stellar-captive-core.cfg
```

(We use CLI parameters over environmental variables as well as values from the earlier sections here to maximize clarity.)

This will start serving *two* endpoints: a Captive Core HTTP server on port 8080, which serves up processed ledgers and can be queried by Horizon, and the underlying Core HTTP endpoint on port 11626 (the default).

### Ingestion Instance
Returning to the Horizon instance that will be doing ingestion, we just need to supply the appropriate URLs and ports. If the above server can be resolved on the `captive-core.local` hostname, running Horizon would look like:

```bash
stellar-horizon \
  --network-passphrase='Test SDF Network ; September 2015' \
  --history-archive-urls='https://history.stellar.org/prd/core-testnet/core_testnet_001' \
  --db-url='postgres://postgres:secret@db.local:5432/horizon?sslmode=disable' \
  --remote-captive-core-url=http://captive-core.local:8080 \
  --stellar-core-url=http://captive-core.local:11626 \
  --port=8001 \
  --ingest=true \
  --enable-captive-core-ingestion=true
```

(Again, we prefer CLI parameters to avoid conflating the `/etc/default/stellar-horizon` we defined earlier for the single-machine case.)


### Serving Instance
This is the simplest instance, requiring none of the ingestion parameters:

```bash
stellar-horizon \
  --network-passphrase='Test SDF Network ; September 2015' \
  --history-archive-urls='https://history.stellar.org/prd/core-testnet/core_testnet_001' \
  --db-url='postgres://postgres:secret@db.local:5432/horizon?sslmode=disable' \
  --remote-captive-core-url=http://captive-core.local:8080 \
  --stellar-core-url=http://captive-core.local:11626 \
  --port=8000
```

At this point, you should be able to hit port 8000 on the above instance and watch the `ingest_latest_ledger` value grow.


# Reingestion
If you need to manually reingest some ledgers (for example, you want history for some ledgers that closed before your asset got issued), you can still do this with Captive Core.

For example, suppose we've ingested from ledger 811520, but would like another 1000 ledgers before it to be ingested as well.

```bash
stellar-horizon-cmd db reingest range 810520 811520
```

TODO: Finish this once Slack thread is resolved.
