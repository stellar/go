**TODO:** This should be merged into `captive_core.md` once @fons and I wrap up our individual portions (or decide how break into separate files; maybe we need a `captive_core/` doc subdirectory).

# Migration 
In this section, we'll discuss migrating existing systems running the [latest](https://github.com/stellar/go/releases/latest) stable version of Horizon ([1.13](https://github.com/stellar/go/releases/tag/horizon-v1.13.0) as of this writing) to the new 2.0 beta. 

**Environment assumptions**:

  - For simplicity, we assume a single-machine Ubuntu setup running both Horizon and Core with a single local PostgreSQL server. The URI in the configs outlined later give further insight into the setup. Loosening this assumption is covered briefly in a [later section](#multi-machine-setup).

  - We assume your machine has enough memory to hold Captive Core's in-memory database (~3GiB), which is a larger memory requirement than a traditional Core setup (which would have an on-disk database).

  - We assume that your node joined the Stellar network recently (that is, it has a few thousand ledgers synced / ingeste), though this doesn't really matter. This assumption can be easily relaxed to any range, but we clarify it here so the chosen ledger ranges below make sense.


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


## Upgrading
At this point, all that is left to do is to:

 - modify the Horizon configuration to enable Captive Core (we will assume it lives in `/etc/default/stellar-horizon`, the default)
 - create a Captive Core configuration stub
 - stop the existing Stellar Core instance
 - restart Horizon


### Configure Horizon
First, add the following lines to the Horizon configuration to enable a Captive Core subprocess:

```bash
echo "STELLAR_CORE_BINARY_PATH=$(which stellar-core)
CAPTIVE_CORE_CONFIG_APPEND_PATH=/etc/default/stellar-captive-core.cfg
ENABLE_CAPTIVE_CORE_INGESTION=1" | sudo tee -a /etc/default/stellar-horizon
```

### Configure Captive Core
Captive Core runs with a trimmed down configuration "stub": at minimum, it must contain enough info to set up a quorum. For example,

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

The rest of the configuration will be generated automagically at runtime.

**Note:** Using your existing Stellar Core configuration will not work (**why not???**). Running Horizon will fail with the following error, or errors like it:

    default: Config from /tmp/captive-stellar-core-38cff455ad3469ec/stellar-core.conf
    default: Got an exception: Failed to parse '/tmp/captive-stellar-core-38cff455ad3469ec/stellar-core.conf' :Key HTTP_PORT already present at line 10 [CommandLine.cpp:1064]


### Restarting Services
Now, we can stop Core (which hopefully doesn't need an explanation) and restart Horizon:

```bash
stellar-horizon-cmd serve
```

The logs should show Captive Core running successfully as a subprocess, and eventually Horizon will be running as usual, except with Captive Core rapidly generating transaction metadata in-memory!


## Multi-Machine Setup
If you plan on running Horizon and Captive Core on separate machines, you'll need to change only a few things. Namely, rather than configuring the `STELLAR_CORE_BINARY` variable, you'll need to point Horizon at the Remote Captive Core instance via `REMOTE_CAPTIVE_CORE_URL`.
