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

## Table of Contents

* [regulated\-assets\-approval\-server](#regulated-assets-approval-server)
  * [Table of Contents](#table-of-contents)
  * [Usage](#usage)
    * [Usage: Migrate](#usage-migrate)
      * [Migration files](#migration-files)
    * [Usage: Serve](#usage-serve)
  * [Account Setup](#account-setup)
    * [GET /friendbot?addr=\{stellar\_address\}](#get-friendbotaddrstellar_address)
  * [API Spec](#api-spec)
    * [POST /tx\-approve](#post-tx-approve)
    * [POST /kyc\-status/\{CALLBACK\_ID\}](#post-kyc-statuscallback_id)
    * [GET /kyc\-status/\{STELLAR\_ADDRESS\_OR\_CALLBACK\_ID\}](#get-kyc-statusstellar_address_or_callback_id)
    * [DELETE /kyc\-status/\{STELLAR\_ADDRESS\}](#delete-kyc-statusstellar_address)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc.go)

## Usage

```sh
$ go install
$ regulated-assets-approval-server --help
SEP-8 Approval Server

Usage:
  regulated-assets-approval-server [command] [flags]
  regulated-assets-approval-server [command]

Available Commands:
  migrate     Run migrations on the database
  serve       Serve the SEP-8 Approval Server

Use "regulated-assets-approval-server [command] --help" for more information about a command.
```

### Usage: Migrate

```sh
$ go install
$ regulated-assets-approval-server migrate --help
Run migrations on the database

Usage:
  regulated-assets-approval-server migrate [up|down] [count] [flags]

Flags:
      --database-url string   Database URL (DATABASE_URL) (default "postgres://localhost:5432/?sslmode=disable")
```

#### Migration files

regulated-assets-approval-server builds the migrations into the binary. If there are any changes to the db(adding, removing or updating tables) generate a new `internal/db/dbmigrate/dbmigrate_generated.go` using the `gogenerate.sh` script located at the root of the repo.

```sh
$ ./gogenerate.sh
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
      --database-url string            Database URL (DATABASE_URL) (default "postgres://localhost:5432/?sslmode=disable")
      --friendbot-payment-amount int   The amount of regulated assets the friendbot will be distributing (FRIENDBOT_PAYMENT_AMOUNT) (default 10000)
      --horizon-url string             Horizon URL used for looking up account details (HORIZON_URL) (default "https://horizon-testnet.stellar.org/")
      --issuer-account-secret string   Secret key of the asset issuer's stellar account. (ISSUER_ACCOUNT_SECRET)
      --network-passphrase string      Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                       Port to listen and serve on (PORT) (default 8000)
      --base-url string                The base url address to this server(BASE_URL)
      --kyc-required-payment-amount-threshold string The amount threshold when KYC is required, may contain decimals and is greater than 0 (KYC_REQUIRED_PAYMENT_AMOUNT_THRESHOLD)(default 500 units)
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

After setting up the issuer account you can fund a stellar account with an initial balance of the regulated asset with our internal `friendbot/?addr={stellar_address}` endpoint.
This endpoint is not part of the official SEP-8 Approval Server spec, it's a debug feature to allow accounts to test sending transactions (payments with the issuer's regulated asset) to the server.

### `GET /friendbot?addr={stellar_address}`

This endpoint sends a payment of 10,000 (this value is configurable) regulated
assets to the provided `addr`. Please be aware the address must first establish
a trustline to the regulated asset in order to receive that payment. You can use
[this
link](https://laboratory.stellar.org/#txbuilder?params=eyJhdHRyaWJ1dGVzIjp7ImZlZSI6IjEwMCIsImJhc2VGZWUiOiIxMDAiLCJtaW5GZWUiOiIxMDAifSwiZmVlQnVtcEF0dHJpYnV0ZXMiOnsibWF4RmVlIjoiMTAwIn0sIm9wZXJhdGlvbnMiOlt7ImlkIjowLCJhdHRyaWJ1dGVzIjp7ImFzc2V0Ijp7InR5cGUiOiJjcmVkaXRfYWxwaGFudW00IiwiY29kZSI6IiIsImlzc3VlciI6IiJ9fSwibmFtZSI6ImNoYW5nZVRydXN0In1dfQ%3D%3D&network=test)
to do that in Stellar Laboratory.

## API Spec
### `POST /tx-approve`

This is the core [SEP-8] endpoint used to validate and process approval/revision/rejection of regulated assets transactions.
Note: The example responses below have set their `base-url` env var to `"https://sep8-base-url.com"`.

**Request:**

```json
{
  "tx": "AAAAAgAAAAA0Nk3++mfFw4Is6OaUJTKe71XNtxdktcjGrPildK84xAAAJxAAAJ3YAAAABwAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAAARllVv+K58Rbwzc/2Ti1IsisLC03udNJblQx2sPLfDygAAAAJNWVVTRAAAAAAAAAAAAAAAqjdTmDnuZm4YrIZ3wQVmVXmWSMO4dLk5dOPzUjWDvIgAAAABKp6IgAAAAAAAAAABdK84xAAAAEACHShDhulyTyvFx9lCU2LjAN9P7g6XqZJ6aNKo/NFb+9awp4pE5soK5cTtahhVzx9RsUcH+FSRmOPu4YEqqBsK"
}
```

**Responses:**

_Revised:_

```json
{
  "status": "revised",
  "message": "Authorization and deauthorization operations were added.",
  "tx": "AAAAAgAAAAA0Nk3++mfFw4Is6OaUJTKe71XNtxdktcjGrPildK84xAAABdwAAJ3YAAAABwAAAAEAAAAAAAAAAAAAAABgXdapAAAAAAAAAAUAAAABAAAAAKo3U5g57mZuGKyGd8EFZlV5lkjDuHS5OXTj81I1g7yIAAAABwAAAAA0Nk3++mfFw4Is6OaUJTKe71XNtxdktcjGrPildK84xAAAAAJNWVVTRAAAAAAAAAAAAAABAAAAAQAAAACqN1OYOe5mbhishnfBBWZVeZZIw7h0uTl04/NSNYO8iAAAAAcAAAAAEZZVb/iufEW8M3P9k4tSLIrCwtN7nTSW5UMdrDy3w8oAAAACTVlVU0QAAAAAAAAAAAAAAQAAAAAAAAABAAAAABGWVW/4rnxFvDNz/ZOLUiyKwsLTe500luVDHaw8t8PKAAAAAk1ZVVNEAAAAAAAAAAAAAACqN1OYOe5mbhishnfBBWZVeZZIw7h0uTl04/NSNYO8iAAAAAEqnoiAAAAAAQAAAACqN1OYOe5mbhishnfBBWZVeZZIw7h0uTl04/NSNYO8iAAAAAcAAAAAEZZVb/iufEW8M3P9k4tSLIrCwtN7nTSW5UMdrDy3w8oAAAACTVlVU0QAAAAAAAAAAAAAAAAAAAEAAAAAqjdTmDnuZm4YrIZ3wQVmVXmWSMO4dLk5dOPzUjWDvIgAAAAHAAAAADQ2Tf76Z8XDgizo5pQlMp7vVc23F2S1yMas+KV0rzjEAAAAAk1ZVVNEAAAAAAAAAAAAAAAAAAAAAAAAATWDvIgAAABAxXindTDbKTpw9B+1aUdTOTE6CUF610A0ZL+ofBVSlcvHYadc3LfO/L4/V22h2FyHNt2ALwncmlEq+3hpojZDDQ=="
}
```

_Rejected:_

```json
{
  "status": "rejected",
  "error": "There is one or more unauthorized operations in the provided transaction."
}
```

_Action Required:_

```json
{
  "status": "action_required",
  "message": "Payments exceeding 500.00 GOAT needs KYC approval. Please provide an email address.",
  "action_url": "https://sep8-base-url.com/kyc-status/cf4fe081-5b38-48b6-86ed-1bcfb7171c7d",
  "action_method": "POST",
  "action_fields": [
    "email_address"
  ]
}
```

### `POST /kyc-status/{CALLBACK_ID}`

This endpoint is used for the extra action after `/tx-approve`, as described in
the SEP-8 [Action Required] section.

Currently an arbitrarily criteria is implemented, email addresses starting with "x" will have the KYC
automatically denied while all other emails will be accepted.

Note: Subsequent KYC attempts with new (valid)emails addresses will approve your account for KYC required transactions.

**Request:**

```json
{
  "email_address": "foo@bar.com"
}
```

**Response:**

```json
{
  "result": "no_further_action_required",
}
```

After the user has been approved or rejected they can POST their transaction to [`POST /tx-approve`](#post-tx-approve) for revision.

If their KYC was rejected they should see a rejection response.
**Response (rejected for emails staring with "x"):**

```json
{
  "status": "rejected",
  "error": "Your KYC was rejected and you're not authorized for operations above 500.00 GOAT."
}
```

### `GET /kyc-status/{STELLAR_ADDRESS_OR_CALLBACK_ID}`

Returns the detail of an account that requested KYC, as well some metadata about
its status.
Note: This functionality is for test/debugging purposes and it's not part of the [SEP-8] spec.

**Response (No KYC Submitted):**

```json
{
  "stellar_address": "GA2DMTP67JT4LQ4CFTUONFBFGKPO6VONW4LWJNOIY2WPRJLUV44MJZOK",
  "callback_id":"e0d9243a-40cf-4baa-9575-913e6c98a12e",
  "created_at": "2021-03-26T09:35:06.907293-03:00",
}
```

**Response (Approved KYC):**

```json
{
  "stellar_address": "GA2DMTP67JT4LQ4CFTUONFBFGKPO6VONW4LWJNOIY2WPRJLUV44MJZOK",
  "callback_id":"e0d9243a-40cf-4baa-9575-913e6c98a12e",
  "email_address": "test@test.com",
  "created_at": "2021-03-26T09:35:06.907293-03:00",
  "kyc_submitted_at": "2021-03-26T14:03:43.314334-03:00",
  "approved_at": "2021-03-26T14:03:43.314334-03:00",
}
```

**Response (Rejected KYC):**

```json
{
  "stellar_address": "GA2DMTP67JT4LQ4CFTUONFBFGKPO6VONW4LWJNOIY2WPRJLUV44MJZOK",
  "callback_id":"e0d9243a-40cf-4baa-9575-913e6c98a12e",
  "email_address": "xtest@test.com",
  "created_at": "2021-03-26T09:35:06.907293-03:00",
  "kyc_submitted_at": "2021-03-26T14:03:43.314334-03:00",
  "rejected_at": "2021-03-26T14:03:43.314334-03:00",
}
```

### `DELETE /kyc-status/{STELLAR_ADDRESS}`

Deletes a stellar account from the list of KYCs. If the stellar address is not
in the database to be deleted the server will return with a `404 - Not Found`.
Note: This functionality is for test/debugging purposes and it's not part of the [SEP-8] spec.

**Response:**

```json
{
  "message": "ok"
}
```

[SEP-8]: https://github.com/stellar/stellar-protocol/blob/7c795bb9abc606cd1e34764c4ba07900d58fe26e/ecosystem/sep-0008.md
[authorization flags]: https://github.com/stellar/stellar-protocol/blob/7c795bb9abc606cd1e34764c4ba07900d58fe26e/ecosystem/sep-0008.md#authorization-flags
[Action Required]: https://github.com/stellar/stellar-protocol/blob/7c795bb9abc606cd1e34764c4ba07900d58fe26e/ecosystem/sep-0008.md#action-required
