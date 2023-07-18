# **Horizon Development Guide**

This document describes how to build Horizon from source, so that you can test and edit the code locally to develop bug fixes and new features.

# **Running Stellar with Docker**

The `docker` folder provides a set of docker files and start script to quickly setup a running Horizon instance.

## **Dependencies**

The only dependency you will need to install is [Docker](https://www.docker.com/products/docker-desktop).

## **Start script**

[start.sh](./start.sh) builds horizon from current source, and then runs docker-compose to start the docker containers with runtime config for horizon, postgres, and optionally core if optional `standalone` network parameter was included.

The script takes one optional parameter which configures the Stellar network used by the docker containers. If no parameter is supplied, the containers will run on the Stellar test network.

`./start.sh pubnet` will run the containers on the Stellar public network.

`./start.sh standalone` will run the containers on a private standalone Stellar network.

The following ports will be exposed:
- Horizon: **8000**
- Horizon-Postgres: **5432**
- Stellar-Core (If `standalone` specified): **11626**
- Stellar-Core-Postgres: **5641**

## **Connecting to the Stellar Public Network**

By default, the Docker Compose file configures Stellar Core to connect to the Stellar test network. If you would like to run the docker containers on the
Stellar public network, run `./start.sh pubnet`. 

To run the containers on a private stand-alone network, run `./start.sh standalone`.
When you run Stellar Core on a private stand-alone network, an account will be created which will hold 100 billion Lumens.
The seed for the account will be emitted in the Stellar Core logs:

```
2020-04-22T18:39:19.248 GD5KD [Ledger INFO] Root account seed: SC5O7VZUXDJ6JBDSZ74DSERXL7W3Y5LTOAMRF7RQRL3TAGAPS7LUVG3L
```

When running Horizon on a private stand-alone network, Horizon will not start ingesting until Stellar Core creates its first history archive snapshot. Stellar Core creates snapshots every 64 ledgers, which means ingestion will be delayed until ledger 64.

When you switch between different networks you will need to clear the Stellar Core and Stellar Horizon databases. You can wipe out the databases by running `docker-compose down --remove-orphans -v`.

## **Using a specific version of Stellar Core**

By default, the Docker Compose file is configured to use version 18 of Protocol and Stellar Core. You want the Core version to be at same level as the version horizon repo expects for ingestion. You can specify optional environment variables from the command shell for stating version overrides for either the docker-compose or start.sh invocations. 
```bash
export PROTOCOL_VERSION="18"
eport CORE_IMAGE="stellar/stellar-core:18" 
export STELLAR_CORE_VERSION="18.1.1-779.ef0f44b44.focal"
```

Example:

Runs Stellar Protocol and Core version 18, for any mode of testnet, standalone, pubnet
```bash
PROTOCOL_VERSION=18 CORE_IMAGE=stellar/stellar-core:18 STELLAR_CORE_VERSION=18.1.1-779.ef0f44b44.focal ./start.sh [standalone|pubnet]
```

# **Installing and Developing Horizon**

## **Docker Installation**

The steps for a Horizon development/contributing cycle are as follows:

1. Use the `start.sh` script to spin up the horizon-postgres and horizon containers. The horizon container will also start an instance of stellar-core to be used as the captive core process for ingestion.
    ```bash
    ./start.sh testnet
    ```

2. Check `localhost:8000` to see if horizon is successfully running and exposed on the port.

3. Now you can proceed with edit-build-run cycles locally. Make sure you have not broken anything first by running the test suite from the `go/services/horizon` directory.
    ```bash
    go test ./...
    ```

4. Once all the tests are passing successfully, you need to stop and rebuild the horizon docker container for the changes to take effect. Go to `services/horizon/docker` and run the following commands:
    ```bash
    docker-compose down
    ./start.sh testnet
    ```
Horizon will be recompiled and re-deployed with your code changes.

Although this process works, it is pretty cumbersome since you need to stop and start the docker containers after each code change. The horizon docker container takes a lot of time to build and this hinders the use of a debugger.

## **Local Installation (Recommended)**

We will now configure a development environment to run Horizon locally without Docker.

### **Building Stellar Core**

Horizon requires a local instance of stellar-core called Captive Core. Since, we are doing this for dev purposes its a good idea to build it from scratch using the latest release of stellar-core. Head over to the [INSTALL.md](https://github.com/stellar/stellar-core/blob/master/INSTALL.md) file for the instructions.

### **Building Horizon**

The following developer tools are required:

- A [Unix-like](https://en.wikipedia.org/wiki/Unix-like) operating system with the common core commands (cp, tar, mkdir, bash, etc.)
- Go (this repository is officially supported on the last two releases of Go)
- [git](https://git-scm.com/) (to check out Horizon's source code)
- [mercurial](https://www.mercurial-scm.org/) (needed for `go-dep`)

1. Fork the repository and clone the fork into any directory you prefer:
   ```bash
   git clone https://github.com/<your-github-username>/go
   ```
2. Change to the directory where the repository is checked out. e.g. `cd go/services/horizon/`.
3. Compile the Horizon binary: `go build -o stellar-horizon && go install`. You should see the resulting `stellar-horizon` executable in `go/services/horizon`.
4. Add the executable to your PATH in your `~/.bashrc` or equivalent, for easy access: `export PATH=$PATH:{absolute-path-to-horizon-folder}`

Open a new terminal. Confirm everything worked by running `stellar-horizon --help` successfully. You should see an informative message listing the command line options supported by Horizon.

### **Run tests**
At this point you should be able to run Horizon's unit tests:
```bash
cd /go/services/horizon/
go test ./...
```

To run the integration tests, you need to set some environment variables:

```bash
export HORIZON_INTEGRATION_TESTS_ENABLED=true 
export HORIZON_INTEGRATION_TESTS_CORE_MAX_SUPPORTED_PROTOCOL=19
export HORIZON_INTEGRATION_TESTS_DOCKER_IMG=stellar/stellar-core:latest
go test -race -timeout 25m -v ./services/horizon/internal/integration/...
```
This will also require a Postgres instance running on port 5432 either locally or exposed through a docker container. Also note that the ``POSTGRES_HOST_AUTH_METHOD`` has been enabled.

### **Database Setup**

Horizon uses a Postgres database backend to record information ingested from an associated Stellar Core. The unit and integration tests will also attempt to reference a Postgres db server at ``localhost:5432`` with trust auth method enabled by default for postgres user.  You can either install the server locally or run any type of docker container that hosts the database server. Here is one example:
```bash
docker run --platform linux/amd64 -d --env POSTGRES_HOST_AUTH_METHOD=trust -p 5432:5432 circleci/postgres:12-alpine
```

Or use the ``docker-compose.yml`` file in the ``docker`` container:
```bash
docker compose -f ./docker/docker-compose.yml up horizon-postgres
```
This starts a Horizon Postgres docker container and exposes it on the port 5432.


### **Setup Debug Configuration in IDE**

#### **Code Debug**
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
         "NETWORK_PASSPHRASE": "Test SDF Network ; September 2015",
         "CAPTIVE_CORE_CONFIG_APPEND_PATH": "${workspaceRoot}/services/horizon/docker/captive-core-testnet.cfg",
         "HISTORY_ARCHIVE_URLS": "https://history.stellar.org/prd/core-testnet/core_testnet_001",
         "INGEST": "true",
         "PER_HOUR_RATE_LIMIT": "0"
     },
     "args": []
 }
```
If all is well, you should see ingest logs written to standard out.

#### **Test Debug**
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
       "HORIZON_INTEGRATION_TESTS_DOCKER_IMG": "stellar/stellar-core:latest"
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

## **Testing Horizon API using Stellar Laboratory**

You can test your Horizon instance with a query like: `http://localhost:8001/transactions?limit=10&order=asc`. However, it's much easier to use the [Stellar Laboratory](https://www.stellar.org/laboratory/) to craft other queries to try out. Select `Custom` Horizon URL and enter `http://localhost:8000`.

Read about the available endpoints and see examples in the [Horizon API reference](https://www.stellar.org/developers/horizon/reference/).

## **Development Cycle**
Congratulations! You can now run the full development cycle to build and test your code.
1. Write code + tests
2. Run tests
3. Compile and Run Horizon: `go install /go/services/horizon && go build /go/services/horizon`. Tip: If you have setup the debug configuration as mentioned above, then it will always build and start Horizon so you do not need to compile it manually.

# **Notes for Developers**

This section contains additional information related to the development of Horizon.

## <a name="regen"></a> **Regenerating generated code**

You can run the following terminal snippet:
```bash
cd go
./gogenerate.sh
```

## <a name="logging"></a> **Logging**

All logging infrastructure is in the `go/services/horizon/internal/log` package.  This package provides "level-based" logging:  Each logging statement has a severity, one of "Debug", "Info", "Warn", "Error" or "Panic".  The Horizon server has a configured level "filter", specified either using the `--log-level` command line flag or the `LOG_LEVEL` environment variable.  When a logging statement is executed, the statements declared severity is checked against the filter and will only be emitted if the severity of the statement is equal or higher severity than the filter.

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

The logging package provides the root logger at `log.DefaultLogger` and the package level funcs such as `log.Info` operate against the default logger.  However, often it is important to include request-specific fields in a logging statement that are not available in the local scope.  For example, it is useful to include an http request's id in every log statement that is emitted by code running on behalf of the request.  This allows for easier debugging, as an operator can filter the log stream to a specific request id and not have to wade through the entirety of the log.

Unfortunately, it is not prudent to thread an `*http.Request` parameter to every downstream subroutine and so we need another way to make that information available.  The idiomatic way to do this in Go is with a context parameter, as describe [on the Go blog](https://blog.golang.org/context).  The logging provides a func to bind a logger to a context using `log.Set` and allows you to retrieve a bound logger using `log.Ctx(ctx)`.  Functions that need to log on behalf of an server request should take a context parameter.

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

