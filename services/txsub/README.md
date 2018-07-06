# Transaction Submission Service


Plug and play transaction submission micro-service. This will ultimately have two modes:

1. horizon proxy mode
2. stellar core mode


## Config

By default this server uses a config file named `txsub.cfg` in the current working directory. This configuration file should be a [TOML](https://github.com/toml-lang/toml) file. The following fields are supported:

* `port` - server listening port
* `horizon_url` - url of the upstream horizon instance for reference and transaction submission
* `network_passphrase` - specify which network to use
* `mode` - specify which mode to use. Currently supports *horizon proxy* with plans to also support *stellar core*

## Example `txsub.cfg`
In this section you can find config examples for the two main ways of setting up a txsub service.

### #1: Horizon Proxy mode

In the case you'll utilize an upstream horizon instance to submit transactions and query data:

```toml
port = 8000
horizon_url = "https://horizon_url.com"
network_passphrase = "Some network passphrase"
mode = "horizon proxy"
```

### #2: Stellar Core mode

*Coming Soon!*

## Usage

```
./txsub [-c=CONFIGPATH]
```

## Building

This service can built from source, provided you have installed the [go tools](https://golang.org/doc/install), by issuing the following command in a terminal:

Given you have a running golang installation, you can build the server with:

```
go get -u github.com/stellar/go/services/txsub
```

After successful completion, you should find `bin/txsub` is present in your configured GOPATH.

## Running tests

```
go test
```