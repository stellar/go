# Horizon Development Guide

This document describes how to build Horizon from source, so that you can test and edit the code locally to develop bug fixes and new features.

## Dependencies
- A [Unix-like](https://en.wikipedia.org/wiki/Unix-like) operating system with the common core commands (cp, tar, mkdir, bash, etc.)
- Go (this repository is officially supported on the last [two releases of Go](https://go.dev/doc/devel/release))
- [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) (to check out Horizon's source code)
- [mercurial](https://www.mercurial-scm.org/) (needed for `go-dep`)
- [Docker](https://www.docker.com/products/docker-desktop)

## The Go Monorepo
All the code for Horizon resides in our Go monorepo.
```bash
git clone https://github.com/go.git
```
If you want to contribute to the project, consider forking the repository and cloning the fork instead.

## Getting Started with Running Horizon
The [start.sh](/services/horizon/docker/start.sh) script builds horizon from current source, and then runs docker-compose to start the docker containers with runtime configs for horizon, postgres, and optionally core if the optional `standalone` network parameter was included. 
The script takes one optional parameter which configures the Stellar network used by the docker containers. If no parameter is supplied, the containers will run on the Stellar test network. Read more about the public and private networks in the [public documentation](https://developers.stellar.org/docs/fundamentals-and-concepts/testnet-and-pubnet#testnet)

`./start.sh pubnet` will run the containers on the Stellar public network.

`./start.sh standalone` will run the containers on a private standalone Stellar network.

`./start.sh testnet` will run the containers on the Stellar test network.

The following ports will be exposed:
- Horizon: **8000**
- Horizon-Postgres: **5432**
- Stellar-Core (If `standalone` specified): **11626**
- Stellar-Core-Postgres (If `standalone` specified): **5641**

Note that when you switch between different networks you will need to clear the Stellar Core and Stellar Horizon databases. You can wipe out the databases by running `docker-compose down --remove-orphans -v`. 

This script is helpful to spin up the services quickly and play around with them. However, for code development it's important to build and install everything locally 

## Developing Horizon Locally
We will now configure a development environment to run Horizon service locally without Docker.

### Building Stellar Core
Horizon requires an instance of stellar-core binary on the same host. This is referred to as the `Captive Core`. Since, we are running horizon for dev purposes, we recommend considering two approaches to get the stellar-core binary, if saving time is top priority and your development machine is on a linux debian o/s, then consider installing the debian package, otherwise the next option available is to compile the core source directly to binary on your machine, refer to [INSTALL.md](https://github.com/stellar/stellar-core/blob/master/INSTALL.md) file for the instructions on both approaches.

### Building Horizon

1. Change to the horizon services directory - `cd go/services/horizon/`.
2. Compile the Horizon binary: `go build -o stellar-horizon && go install`. You should see the resulting `stellar-horizon` executable in `go/services/horizon`.
3. Add the executable to your PATH in your `~/.bashrc` or equivalent, for easy access: `export PATH=$PATH:{absolute-path-to-horizon-folder}`

Open a new terminal. Confirm everything worked by running `stellar-horizon --help` successfully. You should see an informative message listing the command line options supported by Horizon.

### Database Setup

Horizon uses a Postgres database backend to record information ingested from an associated Stellar Core. The unit and integration tests will also attempt to reference a Postgres db server at ``localhost:5432`` with trust auth method enabled by default for ``postgres`` user.  You can either install the server locally or run any type of docker container that hosts the database server. We recommend using the [docker-compose.yml](/services/horizon/docker/docker-compose.yml) file in the ``docker`` folder:
```bash
docker-compose -f ./docker/docker-compose.yml up horizon-postgres
```
This starts a Horizon Postgres docker container and exposes it on the port 5432. Note that while Horizon will run locally, it's PostgresQL db will run in docker.

To shut down all docker containers and free up resources, run the following command:
```bash
docker-compose -f ./docker/docker-compose.yml down
```

### Run tests
At this point you should be able to run Horizon's unit tests:
```bash
cd go/services/horizon/
go test ./...
```

To run the integration tests, you need to set some environment variables:

```bash
export HORIZON_INTEGRATION_TESTS_ENABLED=true 
export HORIZON_INTEGRATION_TESTS_CORE_MAX_SUPPORTED_PROTOCOL=19
export HORIZON_INTEGRATION_TESTS_DOCKER_IMG=stellar/stellar-core:19.11.0-1323.7fb6d5e88.focal
go test -race -timeout 25m -v ./services/horizon/internal/integration/...
```
Note that this will also require a Postgres instance running on port 5432 either locally or exposed through a docker container. Also note that the ``POSTGRES_HOST_AUTH_METHOD`` has been enabled.

### Setup Debug Configuration in IDE

#### Code Debug
Add a debug configuration in your IDE to attach a debugger to the local Horizon process and set breakpoints in your code. Here is an example configuration for VS Code:

```json
 {
     "name": "Horizon Debugger",
     "type": "go",
     "request": "launch",
     "mode": "debug",
     "program": "${workspaceRoot}/services/horizon/main.go",
     "env": {
         "DATABASE_URL": "postgres://postgres@localhost:5432/horizon?sslmode=disable",
         "CAPTIVE_CORE_CONFIG_APPEND_PATH": "./ingest/ledgerbackend/configs/captive-core-testnet.cfg",
         "HISTORY_ARCHIVE_URLS": "https://history.stellar.org/prd/core-testnet/core_testnet_001,https://history.stellar.org/prd/core-testnet/core_testnet_002",
         "NETWORK_PASSPHRASE": "Test SDF Network ; September 2015",
         "PER_HOUR_RATE_LIMIT": "0"
     },
     "args": []
 }
```
If all is well, you should see ingest logs written to standard out. You can read more about configuring the different environment variables in [Configuring](https://developers.stellar.org/docs/run-api-server/configuring) section of our public documentation.

#### Test Debug
You can also use a similar configuration to debug the integration and unit tests. For e.g. here is a configuration for debugging the ```TestFilteringAccountWhiteList``` integration test.
```json
{
   "name": "Debug Test Function",
   "type": "go",
   "request": "launch",
   "mode": "test",
   "program": "${workspaceRoot}/services/horizon/internal/integration",
   "env": {
       "HORIZON_INTEGRATION_TESTS_ENABLED": "true",
       "HORIZON_INTEGRATION_TESTS_CORE_MAX_SUPPORTED_PROTOCOL": "19",
       "HORIZON_INTEGRATION_TESTS_DOCKER_IMG": "stellar/stellar-core:19.11.0-1323.7fb6d5e88.focal"
   },
   "args": [
       "-test.run",
       "TestFilteringAccountWhiteList",
       "-test.timeout",
       "5m",
       "./..."
   ],
   "showLog": true
}
```

## Testing Horizon API using Stellar Laboratory

You can test your Horizon instance with a query like: `http://localhost:8000/transactions?limit=10&order=asc`. However, it's much easier to use the [Stellar Laboratory](https://www.stellar.org/laboratory/) to craft other queries to try out. Select `Custom` Horizon URL and enter `http://localhost:8000`.

Read about the available endpoints and see examples in the [Horizon API reference](https://www.stellar.org/developers/horizon/reference/).

# **Notes for Developers**

This section contains additional information related to the development of Horizon.

## Configuring a Standalone Stellar-Core

By default, the [docker-compose.yml](/services/horizon/docker/docker-compose.yml) will configure Horizon with captive core ingestion to use the test network.

To run the containers on a private stand-alone network, run `./start.sh standalone`.
When you run Stellar Core on a stand-alone network, a root account will be created by default. It will have a balance of 100 billion Lumens and the following key pair:
```
Root Public Key: GBZXN7PIRZGNMHGA7MUUUF4GWPY5AYPV6LY4UV2GL6VJGIQRXFDNMADI
Root Secret Key: SC5O7VZUXDJ6JBDSZ74DSERXL7W3Y5LTOAMRF7RQRL3TAGAPS7LUVG3L
```
When running Horizon on a private stand-alone network, Horizon will not start ingesting until Stellar Core creates its first history archive snapshot. Stellar Core creates snapshots every 64 ledgers, which means ingestion will be delayed until ledger 64.

### Accelerated network for testing

You can increase the speed of tests by adding `ARTIFICIALLY_ACCELERATE_TIME_FOR_TESTING=true` in the [captive-core-standalone.cfg](/services/horizon/docker/captive-core-standalone.cfg).

And finally setting the checkpoint frequency for horizon:
```bash
export CHECKPOINT_FREQUENCY=8
```

This modification causes the standalone network to close ledgers every 1 second and create a checkpoint once every 8 ledgers, 
deviating from the default timing of ledger closing after 5 seconds and creating a checkpoint once every 64 ledgers. Please note that 
this customization is only applicable when running a standalone network, as it requires changes to the captive-core configuration.

## Using a specific version of Stellar Core

By default, the Docker Compose file is configured to use version 19 of Protocol and Stellar Core. You can specify optional environment variables from the command shell for stating version overrides for either the docker-compose or start.sh invocations.
```bash
export PROTOCOL_VERSION="19"
export CORE_IMAGE="stellar/stellar-core:19.11.0-1323.7fb6d5e88.focal" 
export STELLAR_CORE_VERSION="19.11.0-1323.7fb6d5e88.focal"
```

Example:

Runs Stellar Protocol and Core version 19, for any mode of testnet, standalone, pubnet
```bash
PROTOCOL_VERSION=19 CORE_IMAGE=stellar/stellar-core:19.11.0-1323.7fb6d5e88.focal STELLAR_CORE_VERSION=19.11.0-1323.7fb6d5e88.focal ./start.sh [standalone|pubnet]
```

## <a name="logging"></a> **Logging**

All logging infrastructure is implemented using the `go/support/log` package.  This package provides "level-based" logging:  Each logging statement has a severity, one of "Debug", "Info", "Warn", "Error" or "Panic".  The Horizon server has a configured level "filter", specified either using the `--log-level` command line flag or the `LOG_LEVEL` environment variable.  When a logging statement is executed, the statements declared severity is checked against the filter and will only be emitted if the severity of the statement is equal or higher severity than the filter.

In addition, the logging subsystem has support for fields: Arbitrary key-value pairs that will be associated with an entry to allow for filtering and additional contextual information.

### **Making logging statements**

Assuming that you've imported the log package, making a simple logging call is just:

```go

log.Info("my log line")
log.Infof("I take a %s", "format string")
```

Adding fields to a statement happens with a call to `WithField` or `WithFields`

```go
log.WithField("pid", 1234).Warn("i'm scared")

log.WithFields(log.F{
	"some_field": 123,
	"second_field": "hello",
}).Debug("here")
```

The return value from `WithField` or `WithFields` is a `*log.Entry`, which you can save to emit multiple logging
statements that all share the same field.  For example, the action system for Horizon attaches a log entry to `action.Log` on every request that can be used to emit log entries that have the request's id attached as a field.

### **Logging and Context**

The logging package in Go provides a default logger, but it may be necessary to include request-specific fields in log statements. This can be achieved by using a context parameter instead of passing a *http.Request to each subroutine. The context can be bound to a logger with log.Set and retrieved with log.Ctx(ctx) to enable logging on behalf of a server request. This approach allows for easier debugging by filtering log streams based on request IDs. The [Go blog](https://blog.golang.org/context) provides more information on using contexts.

Here's an example of using context:

```go

// create a new sublogger
sub := log.WithField("val", 1)

// bind it to a context
ctx := log.Set(context.Background(), sub)

log.Info("no fields on this statement")
log.Ctx(ctx).Info("This statement will use the sub logger")
```

### **Adding migrations**
1. Add your migration to `go/services/horizon/internal/db2/schema/migrations/` using the same name nomenclature as other migrations.
2. After creating you migration, run `./gogenerate.sh` from the root folder to regenerate the code.

### **Code Formatting**
Some basic code formatting is required for contributions. Run the following scripts included in this repo to perform these tasks. Some of this formatting can be done automatically by IDE's also:
```bash
./gofmt.sh
./govet.sh
./staticcheck.sh
```
