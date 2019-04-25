# txnbuild

`txnbuild` is a [Stellar SDK](https://www.stellar.org/developers/reference/), implemented in [Go](https://golang.org/). It provides a reference implementation of the complete [set of operations](https://www.stellar.org/developers/guides/concepts/list-of-operations.html) that compose [transactions](https://www.stellar.org/developers/guides/concepts/transactions.html) for the Stellar distributed ledger.

This project is maintained by the Stellar Development Foundation.

```
TODO: SHORT EXAMPLE GOES HERE
```

## Getting Started
This library is aimed at developers building Go applications on top of the [Stellar network](https://www.stellar.org/). Transactions constructed by this library may be submitted to any Horizon instance for processing onto the ledger, using any Stellar SDK client. The recommended client for Go programmers is [horizonclient](https://github.com/stellar/go/tree/master/exp/clients/horizon). Together, these two libraries provide a complete Stellar SDK.

* The [txnbuild API reference](https://godoc.org/github.com/stellar/go/exp/txnbuild).
* The [horizonclient API reference](https://godoc.org/github.com/stellar/go/clients/horizonclient).

### Prerequisites
* Go 1.10 or greater

### Installing
* Download the Stellar Go monorepo: `git clone git@github.com:stellar/go.git`
* Enter the source directory: `cd $GOPATH/src/github.com/stellar/go`
* Download external dependencies: `dep ensure -v`

## Running the tests
Run the unit tests from the package directory: `go test`

## Contributing
Please read [CONTRIBUTING](../../CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## License
This project is licensed under the Apache License - see the [LICENSE](../../LICENSE-APACHE.txt) file for details.
