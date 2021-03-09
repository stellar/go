# regulated-assets-approval-server-demo

```
Varsion: v0.0.1
Status: unreleased
```

This is a [SEP-8] Approval Server reference implementation based on SEP-8 v1.6.1
intended for **testing only**. It is being concieved to:

1. Be used as an example of how regulated assets transactions can be validated
   and revised by an anchor.
2. Serve as a demo server where wallets can test and validate their SEP-8
   implementation.

Please be aware this is a demo and its code is not mean to be used in a
production application.

## Usage

```
$ go install
$ regulated-assets-approval-server-demo --help
SEP-8 Approval Server Demo

Usage:
  regulated-assets-approval-server-demo [command] [flags]
  regulated-assets-approval-server-demo [command]

Available Commands:
  serve       Run the SEP-8 Approval Server Demo

Use "regulated-assets-approval-server-demo [command] --help" for more information about a command.
```

## Usage: Serve

```
$ go install
$ regulated-assets-approval-server-demo serve --help
Run the SEP-8 Approval Server Demo

Usage:
  regulated-assets-approval-server-demo serve [flags]

Flags:
      --horizon-url string          Horizon URL used for looking up account details (HORIZON_URL) (default "https://horizon-testnet.stellar.org/")
      --network-passphrase string   Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                    Port to listen and serve on (PORT) (default 8000)
```


[SEP-8]: https://github.com/stellar/stellar-protocol/blob/7c795bb9abc606cd1e34764c4ba07900d58fe26e/ecosystem/sep-0008.md