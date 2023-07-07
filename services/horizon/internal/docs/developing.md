# **Horizon Development Guide**

This document describes how to build Horizon from source, so that you can test and edit the code locally to develop bug fixes and new features.

# **Running Stellar with Docker**

Files related to docker and docker-compose
* `Dockerfile` and `Makefile` - used to build the official, package-based docker image for stellar-horizon
* `Dockerfile.dev` - used with docker-compose

## **Dependencies**

The only dependency you will need to install is [Docker](https://www.docker.com/products/docker-desktop).

## **Start script**

[start.sh](./start.sh) will setup the env file and run docker-compose to start the Stellar docker containers. Feel free to use this script, otherwise continue with the next two steps.

The script takes one optional parameter which configures the Stellar network used by the docker containers. If no parameter is supplied, the containers will run on the Stellar test network.

`./start.sh pubnet` will run the containers on the Stellar public network.

`./start.sh standalone` will run the containers on a private standalone Stellar network.

## **Run docker-compose**

Run the following command to start all the Stellar docker containers:

```
docker-compose up -d --build
```

Horizon will be exposed on port 8000. Stellar Core will be exposed on port 11626. The Stellar Core postgres instance will be exposed on port 5641.
The Horizon postgres instance will be exposed on port 5432.

## **Connecting to the Stellar Public Network**

By default, the Docker Compose file configures Stellar Core to connect to the Stellar test network. If you would like to run the docker containers on the
Stellar public network, run `docker-compose -f docker-compose.yml -f docker-compose.pubnet.yml up -d --build`. 

To run the containers on a private stand-alone network, run `docker-compose -f docker-compose.yml -f docker-compose.standalone.yml up -d --build`.
When you run Stellar Core on a private stand-alone network, an account will be created which will hold 100 billion Lumens.
The seed for the account will be emitted in the Stellar Core logs:

```
2020-04-22T18:39:19.248 GD5KD [Ledger INFO] Root account seed: SC5O7VZUXDJ6JBDSZ74DSERXL7W3Y5LTOAMRF7RQRL3TAGAPS7LUVG3L
```

When running Horizon on a private stand-alone network, Horizon will not start ingesting until Stellar Core creates its first history archive snapshot. Stellar Core creates snapshots every 64 ledgers, which means ingestion will be delayed until ledger 64.

When you switch between different networks you will need to clear the Stellar Core and Stellar Horizon databases. You can wipe out the databases by running `docker-compose down --remove-orphans -v`.

## **Using a specific version of Stellar Core**

By default the Docker Compose file is configured to use version 18 of Protocol and Stellar Core. You want the Core version to be at same level as the version horizon repo expects for ingestion. You can specify optional environment variables from the command shell for stating version overrides for either the docker-compose or start.sh invocations. 

PROTOCOL_VERSION=18                              // the Stellar Protocol version number
CORE_IMAGE=stellar/stellar-core:18               // the docker hub image:tag 
STELLAR_CORE_VERSION=18.1.1-779.ef0f44b44.focal  // the apt deb package version from apt.stellar.org

Example:

Runs Stellar Protocol and Core version 18, for any mode of testnet, standalone, pubnet
```PROTOCOL_VERSION=18 CORE_IMAGE=stellar/stellar-core:18 STELLAR_CORE_VERSION=18.1.1-779.ef0f44b44.focal ./start.sh [standalone|pubnet]```

# **Installing and Developing Horizon**

## **Docker Installation**

The steps for a Horizon development/contributing cycle are as follows:

1. Use the start.sh script to spin up the horizon-postgres and horizon containers. The horizon container will also have its own stellar-core running in it.
    ```
    ./start.sh testnet
    ```

2. Check `localhost:8000` to see if horizon is successfully running and exposed on the port.

3. Now you can go ahead and make the required code changes. Make sure you have not broken anything by running the test suite from the `go/services/horizon` directory.
    ```
    go test ./...
    ```

4. Once all the tests are passing successfully, you need to stop and rebuild the horizon docker container for the changes to take effect. Go to `services/horizon/docker` and run the following commands:
    ```
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

1. Set your [GOPATH](https://github.com/golang/go/wiki/GOPATH) environment variable, if you haven't already. The default `GOPATH` is `$HOME/go`. When building any Go package or application the binaries will be installed by default to `$GOPATH/bin`.
2. Fork the repository and clone the fork into any directory you prefer:
   ```
   git clone https://github.com/stellar/go
   ```
   Or if you prefer to develop inside `GOPATH` check it out to `$GOPATH/src/github.com/stellar/go`:
   ```
   git clone https://github.com/stellar/go $GOPATH/src/github.com/stellar/go
   ```
   If developing inside `GOPATH` set the `GO111MODULE=on` environment variable to turn on Modules for managing dependencies. See the repository [README](../../../../README.md#dependencies) for more information.
3. Change to the directory where the repository is checked out. e.g. `cd go`, or if developing inside the `GOPATH`, `cd $GOPATH/src/github.com/stellar/go`.
4. Compile the Horizon binary: `go install ./services/horizon`. You should see the resulting `horizon` executable in `$GOPATH/bin`.
5. Add Go binaries to your PATH in your `bashrc` or equivalent, for easy access: `export PATH=${GOPATH//://bin:}/bin:$PATH`

Open a new terminal. Confirm everything worked by running `horizon --help` successfully. You should see an informative message listing the command line options supported by Horizon.

### **Run tests**
At this point you should be able to run Horizon's unit tests:
```
cd /go/services/horizon
go test ./...
```

### **Database Setup**

Horizon uses a Postgres database backend to store test fixtures and record information ingested from an associated Stellar Core. You can either set it up locally or use the docker container provided in `services/horizon/docker` container.

#### **Local Setup**
1. Install [PostgreSQL 9.6+](https://www.postgresql.org/).
2. Run `createdb horizon` to initialise an empty database for Horizon's use.
3. Run `horizon db init --db-url postgres://localhost/horizon` to install Horizon's database schema.

Using this method, you may run into some potential database issues:

- Depending on your installation's defaults, you may need to configure a Postgres DB user with appropriate permissions for Horizon to access the database you created. Refer to the [Postgres documentation](https://www.postgresql.org/docs/current/sql-createuser.html) for details. Note: Remember to restart the Postgres server after making any changes to `pg_hba.conf` (the Postgres configuration file), or your changes won't take effect!
- Make sure you pass the appropriate database name and user (and port, if using something non-standard) to Horizon using `--db-url`. One way is to use a Postgres URI with the following form: `postgres://USERNAME:PASSWORD@localhost:PORT/DB_NAME`.
- If you get the error `connect failed: pq: SSL is not enabled on the server`, add `?sslmode=disable` to the end of the Postgres URI to allow connecting without SSL. 
If you get the error `zsh: no matches found: postgres://localhost/horizon_dev?sslmode=disable`, wrap the url with single quotes `horizon db init --db-url 'postgres://localhost/horizon_dev?sslmode=disable'`
- If your server is responding strangely, and you've exhausted all other options, reboot the machine. On some systems `service postgresql restart` or equivalent may not fully reset the state of the server.

#### **Docker Setup (Recommended)**
This is a much easier and recommended way of setting up the Postgres db. Just spin up the `horizon-postgres` docker container:
```
docker-compose -f ./services/horizon/docker/docker-compose.yml up horizon-postgres
```
This starts a Horizon Postgres docker container and exposes it on the port 5432.


### **Setup Debug Configuration in IDE**

Add a debug configuration in your IDE to attach a debugger to the local Horizon process and set breakpoints in your code. Here is an example configuration for VS Code:

```
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

## **Testing Horizon API using Stellar Laboratory**

You can test your Horizon instance with a query like: http://localhost:8001/transactions?limit=10&order=asc. However, its much easier to use the [Stellar Laboratory](https://www.stellar.org/laboratory/) to craft other queries to try out. Select `Custom` Horizon URL and enter `http://localhost:8000`.

Read about the available endpoints and see examples in the [Horizon API reference](https://www.stellar.org/developers/horizon/reference/).

## **Development Cycle**
Congratulations! You can now run the full development cycle to build and test your code.
1. Write code + tests
2. Run tests
3. Compile and Run Horizon: `go install /go/services/horizon && go build /go/services/horizon`. Tip: If you have setup the debug configuration as mentioned above, then it will always build and start Horizon so you do not need to compile it manually.


Refer to our [Developer Notes](DEVELOPER_NOTES.md) for details on writing tests and logging.