---
title: Frontier Development Guide
---
## Frontier Development Guide

This document describes how to build Frontier from source, so that you can test and edit the code locally to develop bug fixes and new features.

If you are just starting with Frontier and want to try it out, consider the [Quickstart Guide](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/quickstart.md) instead. For information about administrating a Frontier instance in production, check out the [Administration Guide](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/admin.md).

## Building Frontier
Building Frontier requires the following developer tools:

- A [Unix-like](https://en.wikipedia.org/wiki/Unix-like) operating system with the common core commands (cp, tar, mkdir, bash, etc.)
- Golang 1.14 or later
- [git](https://git-scm.com/) (to check out Frontier's source code)
- [mercurial](https://www.mercurial-scm.org/) (needed for `go-dep`)

1. Set your [GOPATH](https://github.com/golang/go/wiki/GOPATH) environment variable, if you haven't already. The default `GOPATH` is `$HOME/go`. When building any Go package or application the binaries will be installed by default to `$GOPATH/bin`.
2. Checkout the code into any directory you prefer:
   ```
   git checkout https://github.com/xdbfoundation/go
   ```
   Or if you prefer to develop inside `GOPATH` check it out to `$GOPATH/src/github.com/xdbfoundation/go`:
   ```
   git checkout https://github.com/xdbfoundation/go $GOPATH/src/github.com/xdbfoundation/go
   ```
   If developing inside `GOPATH` set the `GO111MODULE=on` environment variable to turn on Modules for managing dependencies. See the repository [README](https://github.com/xdbfoundation/go/blob/master/README.md#dependencies) for more information.
3. Change to the directory where the repository is checked out. e.g. `cd go`, or if developing inside the `GOPATH`, `cd $GOPATH/src/github.com/xdbfoundation/go`.
4. Compile the Frontier binary: `go install ./services/frontier`. You should see the resulting `frontier` executable in `$GOPATH/bin`.
5. Add Go binaries to your PATH in your `bashrc` or equivalent, for easy access: `export PATH=${GOPATH//://bin:}/bin:$PATH`

Open a new terminal. Confirm everything worked by running `frontier --help` successfully. You should see an informative message listing the command line options supported by Frontier.

## Set up Frontier's database
Frontier uses a Postgres database backend to store test fixtures and record information ingested from an associated DigitalBits Core. To set this up:
1. Install [PostgreSQL](https://www.postgresql.org/).
2. Run `createdb frontier_dev` to initialise an empty database for Frontier's use.
3. Run `frontier db init --db-url postgres://localhost/frontier_dev` to install Frontier's database schema.

### Database problems?
1. Depending on your installation's defaults, you may need to configure a Postgres DB user with appropriate permissions for Frontier to access the database you created. Refer to the [Postgres documentation](https://www.postgresql.org/docs/current/sql-createuser.html) for details. Note: Remember to restart the Postgres server after making any changes to `pg_hba.conf` (the Postgres configuration file), or your changes won't take effect!
2. Make sure you pass the appropriate database name and user (and port, if using something non-standard) to Frontier using `--db-url`. One way is to use a Postgres URI with the following form: `postgres://USERNAME:PASSWORD@localhost:PORT/DB_NAME`.
3. If you get the error `connect failed: pq: SSL is not enabled on the server`, add `?sslmode=disable` to the end of the Postgres URI to allow connecting without SSL.
4. If your server is responding strangely, and you've exhausted all other options, reboot the machine. On some systems `service postgresql restart` or equivalent may not fully reset the state of the server.

## Run tests
At this point you should be able to run Frontier's unit tests:
```bash
cd $GOPATH/src/github.com/xdbfoundation/go/services/frontier
go test ./...
```

## Set up DigitalBits Core
Frontier provides an API to the DigitalBits network. It does this by ingesting data from an associated `digitalbits-core` instance. Thus, to run a full Frontier instance requires a `digitalbits-core` instance to be configured, up to date with the network state, and accessible to Frontier. Frontier accesses `digitalbits-core` through both an HTTP endpoint and by connecting directly to the `digitalbits-core` Postgres database.

The simplest way to set up DigitalBits Core is using the [DigitalBits Quickstart Docker Image](https://github.com/digitalbits/docker-digitalbits-core-frontier). This is a Docker container that provides both `digitalbits-core` and `frontier`, pre-configured for testing.

1. Install [Docker](https://www.docker.com/get-started).
2. Verify your Docker installation works: `docker run hello-world`
3. Create a local directory that the container can use to record state. This is helpful because it can take a few minutes to sync a new `digitalbits-core` with enough data for testing, and because it allows you to inspect and modify the configuration if needed. Here, we create a directory called `digitalbits` to use as the persistent volume: `cd $HOME; mkdir digitalbits`
4. Download and run the DigitalBits Quickstart container:

```bash
docker run --rm -it -p "8000:8000" -p "11626:11626" -p "11625:11625" -p"8002:5432" -v $HOME/digitalbits:/opt/digitalbits --name digitalbits digitalbits/quickstart --testnet
```

In this example we run the container in interactive mode. We map the container's Frontier HTTP port (`8000`), the `digitalbits-core` HTTP port (`11626`), and the `digitalbits-core` peer node port (`11625`) from the container to the corresponding ports on `localhost`. Importantly, we map the container's `postgresql` port (`5432`) to a custom port (`8002`) on `localhost`, so that it doesn't clash with our local Postgres install.
The `-v` option mounts the `digitalbits` directory for use by the container. See the [Quickstart Image documentation](https://github.com/digitalbits/docker-digitalbits-core-frontier) for a detailed explanation of these options.

5. The container is running both a `digitalbits-core` and a `frontier` instance. Log in to the container and stop Frontier:
```bash
docker exec -it digitalbits /bin/bash
supervisorctl
stop frontier
```

## Check DigitalBits Core status
DigitalBits Core takes some time to synchronise with the rest of the network. The default configuration will pull roughly a couple of day's worth of ledgers, and may take 15 - 30 minutes to catch up. Logs are stored in the container at `/var/log/supervisor`. You can check the progress by monitoring logs with `supervisorctl`:
```bash
docker exec -it digitalbits /bin/bash
supervisorctl tail -f digitalbits-core
```

You can also check status by looking at the HTTP endpoint, e.g. by visiting http://localhost:11626 in your browser.

## Connect Frontier to DigitalBits Core
You can connect Frontier to `digitalbits-core` at any time, but Frontier will not begin ingesting data until `digitalbits-core` has completed its catch-up process.

Now run your development version of Frontier (which is outside of the container), pointing it at the `digitalbits-core` running inside the container:

```bash
frontier --db-url="postgres://localhost/frontier_dev" --digitalbits-core-db-url="postgres://digitalbits:postgres@localhost:8002/core" --digitalbits-core-url="http://localhost:11626" --port 8001 --network-passphrase "TestNet Global DigitalBits Network ; December 2020" --ingest
```

If all is well, you should see ingest logs written to standard out. You can test your Frontier instance with a query like: http://localhost:8001/transactions?limit=10&order=asc. Use the [DigitalBits Laboratory](https://developers.digitalbits.io/lab/) to craft other queries to try out,
and read about the available endpoints and see examples in the [Frontier API reference](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/readme.md).

## The development cycle
Congratulations! You can now run the full development cycle to build and test your code.
1. Write code + tests
2. Run tests
3. Compile Frontier: `go install github.com/xdbfoundation/go/services/frontier`
4. Run Frontier (pointing at your running `digitalbits-core`)
5. Try Frontier queries

Check out the [DigitalBits Contributing Guide](https://github.com/digitalbits/docs/blob/master/CONTRIBUTING.md) to see how to contribute your work to the DigitalBits repositories. Once you've got something that works, open a pull request, linking to the issue that you are resolving with your contribution. We'll get back to you as quickly as we can.
