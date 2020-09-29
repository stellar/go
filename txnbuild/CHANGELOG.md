# Changelog

All notable changes to this project will be documented in this
file.  This project adheres to [Semantic Versioning](http://semver.org/).

## [v4.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v4.0.0) - 2020-09-29

Added support for the new operations in [Protocol 14](https://github.com/stellar/go/issues/3035). Now it is possible to:
* Create and claim claimable balance operations (see [CAP-23](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0023.md)) with the `[Create|Claim]ClaimableBalance` structures and their associated helpers
* Begin/end sponsoring future reserves for other accounts (see [CAP-33](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0033.md)) with the `[Begin|End]SponsoringFutureReserves` operations
* Revoke sponsorships of various objects with the `RevokeSponsorship` operation (see [CAP-33](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0033.md)).

Also:
* Added support for Go 1.15.
### Breaking changes

* Dropped support for Go 1.13.
* Add support for SEP-10 v2.0.0.
  * Replace `BuildChallengeTx`'s `anchorName` parameter with `homeDomain`.
  * Add `homeDomain` parameter to `ReadChallengeTx`, `VerifyChallengeTxThreshold`, and `VerifyChallengeTxSigners`.

## [v3.2.0](https://github.com/stellar/go/releases/tag/horizonclient-v3.2.0) - 2020-06-18

* `txnbuild` now generates V1 transaction envelopes which are only supported by Protocol 13 ([#2640](https://github.com/stellar/go/pull/2640))
* Add `ToXDR()` functions for `Transaction` and `FeeBumpTransaction` instances which return xdr transaction envelopes without errors ([#2651](https://github.com/stellar/go/pull/2651))

## [v3.1.0](https://github.com/stellar/go/releases/tag/horizonclient-v3.1.0) - 2020-05-14

* Fix bug which occurs when parsing xdr offers with prices that require more than 7 decimals of precision ([#2588](https://github.com/stellar/go/pull/2588))
* Add `AddSignatureBase64` function to both `Transaction` and `FeeBumpTransaction` objects for adding a base64-encoded signature. [#2586](https://github.com/stellar/go/pull/2586)

## [v3.0.1](https://github.com/stellar/go/releases/tag/horizonclient-v3.0.1) - 2020-05-11

* Fix bug which occurs when parsing transactions with manage data operations containing nil values ([#2573](https://github.com/stellar/go/pull/2573))

## [v3.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v3.0.0) - 2020-04-28

### Breaking changes

* The `Account` interface has been extended to include `GetSequenceNumber() (int64, error)`. Also, `IncrementSequenceNumber()` now returns an `(int64, error)` pair instead of a `(xdr.SequenceNumber, error)` pair.
* Refactor workflow for creating and signing transactions. Previously, you could create a transaction envelope by populating a `Transaction` instance and calling the `Build()` function on the `Transaction` instance.

`Transaction` is now an opaque type which has accessor functions like `SourceAccount() SimpleAccount`, `Memo() Memo`, etc. The motivation behind this change is to make `Transaction` more immutable. Here is an example of how to use the new transaction type:
```go
	kp := keypair.MustParse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := txnbuild.Payment{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       NativeAsset{},
	}

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &sourceAccount,
			// If IncrementSequenceNum is true, NewTransaction() will call `sourceAccount.IncrementSequenceNumber()`
			// to obtain the sequence number for the transaction.
			// If IncrementSequenceNum is false, NewTransaction() will call `sourceAccount.GetSequenceNumber()`
			// to obtain the sequence number for the transaction.
			IncrementSequenceNum: true,
			Operations:           []Operation{&op},
			BaseFee:              MinBaseFee,
			Timebounds:           NewInfiniteTimeout(),
		},
	)
	check(err)

	tx, err = tx.Sign(network.TestNetworkPassphrase, kp.(*keypair.Full))
```

* `TransactionFromXDR` now has the following signature `TransactionFromXDR(txeB64 string) (*GenericTransaction, error)`. A `GenericTransaction` is a container which can be unpacked into either a `Transaction` or a `FeeBumpTransaction`.
* `BuildChallengeTx` now returns a `Transaction` instance instead of the base 64 string encoding of the SEP 10 challenge transaction.
* `VerifyChallengeTx` has been removed. Use `VerifyChallengeTxThreshold` or `VerifyChallengeTxSigners` instead.

### Add

* Add `NewFeeBumpTransaction(params FeeBumpTransactionParams) (*FeeBumpTransaction, error)` function for creating [fee bump transactions](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0015.md). Note that fee bump transactions will only be accepted by Stellar Core once Protocol 13 is enabled.

### Updates

* `AllowTrust` supports [CAP0018](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0018.md) Fine-Grained Control of Authorization by exposing a `AuthorizeToMaintainLiabilities` boolean field.
* `ReadChallengeTx` will reject any challenge transactions which are fee bump transactions.
* `ReadChallengeTx` will reject any challenge transactions which contain a [MuxedAccount](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0027.md) with a memo ID.

### Remove

* Dropped support for Go 1.12.

## [v1.5.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.5.0) - 2019-10-09

* Dropped support for Go 1.10, 1.11.
* Add support for stellar-core [protocol 12](https://github.com/stellar/stellar-core/releases/tag/v12.0.0), which implements [CAP-0024](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0024.md) ("Make PathPayment Symmetrical"). ([#1737](https://github.com/stellar/go/issues/1737)).
* **Deprecated:** Following CAP-0024, the operation `txnbuild.PathPayment` is now deprecated in favour of [`txnbuild.PathPaymentStrictReceive`](https://godoc.org/github.com/stellar/go/txnbuild#PathPaymentStrictReceive), and will be removed in a future release. This is a rename - the new operation behaves identically to the old one. Client code should be updated to use the new operation.
* **Add:** New operation [`txnbuild.PathPaymentStrictSend`](https://godoc.org/github.com/stellar/go/txnbuild#PathPaymentStrictSend) allows a path payment to be made where the amount sent is specified, and the amount received can vary.

## [v1.4.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.4.0) - 2019-08-09

* Add `BuildChallengeTx` function for building [SEP-10](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0010.md) challenge transaction([#1466](https://github.com/stellar/go/issues/1466)).
* Add `VerifyChallengeTx` method for verifying [SEP-10](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0010.md) challenge transaction([#1530](https://github.com/stellar/go/issues/1530)).
* Add `TransactionFromXDR` function for building `txnbuild.Transaction` struct from a  base64 XDR transaction envelope[#1329](https://github.com/stellar/go/issues/1329).
* Fix bug that allowed multiple calls to `Transaction.Build` increment the number of operations in a transaction [#1448](https://github.com/stellar/go/issues/1448).
* Add `Transaction.SignWithKeyString` helper method for signing transactions using secret keys as strings.([#1564](https://github.com/stellar/go/issues/1564))


## [v1.3.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.3.0) - 2019-07-08

* Add support for getting the hex-encoded transaction hash with `Transaction.HashHex` method.
* `TransactionEnvelope` is now available after building a transaction(`Transaction.Build`). Previously, this was only available after signing a transaction. ([#1376](https://github.com/stellar/go/pull/1376))
* Add support for getting the `TransactionEnvelope` struct with `Transaction.TxEnvelope` method ([#1415](https://github.com/stellar/go/issues/1415)).
* `AllowTrust` operations no longer requires the asset issuer, only asset code is required ([#1330](https://github.com/stellar/go/issues/1330)).
* `Transaction.SetDefaultFee` method is deprecated and will be removed in the next major release ([#1221](https://github.com/stellar/go/issues/1221)).
* `Transaction.TransactionFee` method has been added to get the fee that will be paid for a transaction.
* `Transaction.SignHashX` method adds support for signing transactions with hash(x) signature types.

## [v1.2.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.2.0) - 2019-05-16

* In addition to account responses from horizon, transactions and operations can now be built with txnbuild.SimpleAccount structs constructed locally ([#1266](https://github.com/stellar/go/issues/1266)).
* Added `MaxTrustlineLimit` which represents the maximum value for a trustline limit ([#1265](https://github.com/stellar/go/issues/1265)).
* ChangeTrust operation with no `Limit` field set now defaults to `MaxTrustlineLimit` ([#1265](https://github.com/stellar/go/issues/1265)).
* Add support for building `ManageBuyOffer` operation ([#1165](https://github.com/stellar/go/issues/1165)).
* Fix bug in ChangeTrust operation builder ([1296](https://github.com/stellar/go/issues/1296)).

## [v1.1.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.1.0) - 2019-02-02

* Support for multiple signatures ([#1198](https://github.com/stellar/go/pull/1198))

## [v1.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.0) - 2019-04-26

* Initial release
