# regulated-assets-approval-server

```sh
Status: unreleased
```

This is a [SEP-8] Approval Server reference implementation based on SEP-8 v1.6.1
intended for **testing only**. It is being conceived to:

1. Be used as an example of how regulated assets transactions can be validated
   and revised by an anchor.
2. Serve as a demo server where wallets can test and validate their SEP-8
   implementation.

## Usage

```sh
$ go install
$ regulated-assets-approval-server --help
SEP-8 Approval Server

Usage:
  regulated-assets-approval-server [command] [flags]
  regulated-assets-approval-server [command]

Available Commands:
  serve       Serve the SEP-8 Approval Server

Use "regulated-assets-approval-server [command] --help" for more information about a command.
```

### Usage: Serve

```sh
$ go install
$ regulated-assets-approval-server serve --help
Serve the SEP-8 Approval Server

Usage:
  regulated-assets-approval-server serve [flags]

Flags:
      --asset-code string              The code of the regulated asset (ASSET_CODE)
      --friendbot-payment-amount int   The amount of regulated assets the friendbot will be distributing (FRIENDBOT_PAYMENT_AMOUNT) (default 10000)
      --horizon-url string             Horizon URL used for looking up account details (HORIZON_URL) (default "https://horizon-testnet.stellar.org/")
      --issuer-account-secret string   Secret key of the asset issuer's stellar account. (ISSUER_ACCOUNT_SECRET)
      --network-passphrase string      Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                       Port to listen and serve on (PORT) (default 8000)
```

## Account Setup

In order to properly use this server for regulated assets, the account whose
secret was added in `--issuer-account-secret (ACCOUNT_ISSUER_SECRET)` needs to
be configured according with SEP-8 [authorization flags] by setting both
`Authorization Required` and `Authorization Revocable` flags. This allows the
issuer to grant and revoke authorization to transact the asset at will.

You can use [this
link](https://laboratory.stellar.org/#txbuilder?params=eyJhdHRyaWJ1dGVzIjp7ImZlZSI6IjEwMCIsImJhc2VGZWUiOiIxMDAiLCJtaW5GZWUiOiIxMDAifSwiZmVlQnVtcEF0dHJpYnV0ZXMiOnsibWF4RmVlIjoiMTAwIn0sIm9wZXJhdGlvbnMiOlt7ImlkIjowLCJhdHRyaWJ1dGVzIjp7InNldEZsYWdzIjozfSwibmFtZSI6InNldE9wdGlvbnMifV19)
to set those flags. Just click the link, fulfill the account address, sequence
number, then the account secret and submit the transaction.

### API Spec

#### `GET /friendbot?addr=GDDIO6SFRD4SJEQFJOSKPIDYTDM7LM4METFBKN4NFGVR5DTGB7H75N5S`

This endpoint sends a payment of 10,000 (this value is configurable) regulated
assets to the provided `addr`. Please be aware the address must first establish
a trustline to the regulated asset in order to receive that payment. You can use
[this
link](https://laboratory.stellar.org/#txbuilder?params=eyJhdHRyaWJ1dGVzIjp7ImZlZSI6IjEwMCIsImJhc2VGZWUiOiIxMDAiLCJtaW5GZWUiOiIxMDAifSwiZmVlQnVtcEF0dHJpYnV0ZXMiOnsibWF4RmVlIjoiMTAwIn0sIm9wZXJhdGlvbnMiOlt7ImlkIjowLCJhdHRyaWJ1dGVzIjp7ImFzc2V0Ijp7InR5cGUiOiJjcmVkaXRfYWxwaGFudW00IiwiY29kZSI6IiIsImlzc3VlciI6IiJ9fSwibmFtZSI6ImNoYW5nZVRydXN0In1dfQ%3D%3D&network=test)
to do that in Stellar Laboratory.

[SEP-8]: https://github.com/stellar/stellar-protocol/blob/7c795bb9abc606cd1e34764c4ba07900d58fe26e/ecosystem/sep-0008.md
[authorization flags]: https://github.com/stellar/stellar-protocol/blob/7c795bb9abc606cd1e34764c4ba07900d58fe26e/ecosystem/sep-0008.md#authorization-flags
