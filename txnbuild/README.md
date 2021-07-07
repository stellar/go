# txnbuild

`txnbuild` is a [DigitalBits SDK](https://developer.digitalbits.io/reference/), implemented in [Go](https://golang.org/). It provides a reference implementation of the complete [set of operations](https://developer.digitalbits.io/guides/concepts/list-of-operations.html) that compose [transactions](https://developer.digitalbits.io/guides/concepts/transactions.html) for the DigitalBits distributed ledger.

This project is maintained by the XDB Foundation.

```golang
    import (
        "log"
        
        "github.com/xdbfoundation/go/clients/frontierclient"
        "github.com/xdbfoundation/go/keypair"
        "github.com/xdbfoundation/go/network"
        "github.com/xdbfoundation/go/txnbuild"
    )
    
    // Make a keypair for a known account from a secret seed
    kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
    
    // Get the current state of the account from the network
    client := frontierclient.DefaultTestNetClient
    ar := frontierclient.AccountRequest{AccountID: kp.Address()}
    sourceAccount, err := client.AccountDetail(ar)
    if err != nil {
        log.Fatalln(err)
    }
    
    // Build an operation to create and fund a new account
    op := txnbuild.CreateAccount{
        Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
        Amount:      "10",
    }
    
    // Construct the transaction that holds the operations to execute on the network
    tx, err := txnbuild.NewTransaction(
        txnbuild.TransactionParams{
            SourceAccount:        &sourceAccount,
            IncrementSequenceNum: true,
            Operations:           []txnbuild.Operation{&op},
            BaseFee:              txnbuild.MinBaseFee,
            Timebounds:           txnbuild.NewTimeout(300),
        },
    )
    if err != nil {
        log.Fatalln(err)
    )
    
    // Sign the transaction
    tx, err = tx.Sign(network.TestNetworkPassphrase, kp.(*keypair.Full))
    if err != nil {
        log.Fatalln(err)
    )
    
    // Get the base 64 encoded transaction envelope
    txe, err := tx.Base64()
    if err != nil {
        log.Fatalln(err)
    }
    
    // Send the transaction to the network
    resp, err := client.SubmitTransactionXDR(txe)
    if err != nil {
        log.Fatalln(err)
    }
```

## Getting Started
This library is aimed at developers building Go applications on top of the [DigitalBits network](https://www.digitalbits.io/). Transactions constructed by this library may be submitted to any Frontier instance for processing onto the ledger, using any DigitalBits SDK client. The recommended client for Go programmers is [frontierclient](https://github.com/xdbfoundation/go/tree/master/clients/frontierclient). Together, these two libraries provide a complete DigitalBits SDK.

* The [txnbuild API reference](https://developer.digitalbits.io/reference/).
* The [frontierclient API reference](https://developer.digitalbits.io/reference/).

An easy-to-follow demonstration that exercises this SDK on the TestNet with actual accounts is also included! See the [Demo](#demo) section below.

### Prerequisites
* Go 1.14 or greater
* [Modules](https://github.com/golang/go/wiki/Modules) to manage dependencies

### Installing
* `go get github.com/xdbfoundation/go/txnbuild`

## Running the tests
Run the unit tests from the package directory: `go test`

## Demo
To see the SDK in action, build and run the demo:
* Enter the demo directory: `cd $GOPATH/src/github.com/xdbfoundation/go/txnbuild/cmd/demo`
* Build the demo: `go build`
* Run the demo: `./demo init`


## Contributing
Please read [Code of Conduct](https://digitalbits.io/community-guidelines/) to understand this project's communication rules.

To submit improvements and fixes to this library, please see [CONTRIBUTING](https://github.com/xdbfoundation/docs/blob/master/CONTRIBUTING.md).

## License
This project is licensed under the Apache License - see the [LICENSE](https://www.apache.org/licenses/LICENSE-2.0) file for details.
