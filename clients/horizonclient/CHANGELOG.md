# Changelog

All notable changes to this project will be documented in this
file.  This project adheres to [Semantic Versioning](http://semver.org/).

## Unreleased

## [v2.1.0](https://github.com/stellar/go/releases/tag/horizonclient-v2.1.0) - 2020-02-24

### Add

- Add `client.StrictReceivePaths` and  `client.StrictSendPaths` ([#2237](https://github.com/stellar/go/pull/2237)).

`client.StrictReceivePaths`:

```go
	client := horizonclient.DefaultPublicNetClient
	// Find paths for XLM->NGN
	pr := horizonclient.PathsRequest{
		DestinationAccount:     "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
		DestinationAmount:      "100",
		DestinationAssetCode:   "NGN",
		DestinationAssetIssuer: "GDZST3XVCDTUJ76ZAV2HA72KYQODXXZ5PTMAPZGDHZ6CS7RO7MGG3DBM",
		DestinationAssetType:   horizonclient.AssetType4,
		SourceAccount:          "GDZST3XVCDTUJ76ZAV2HA72KYQODXXZ5PTMAPZGDHZ6CS7RO7MGG3DBM",
	}
	paths, err := client.StrictReceivePaths(pr)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(paths)
```

`client.StrictSendPaths`:

```go
	client := horizonclient.DefaultPublicNetClient
	// Find paths for USD->EUR
	pr := horizonclient.StrictSendPathsRequest{
		SourceAmount:      "20",
		SourceAssetCode:   "USD",
		SourceAssetIssuer: "GDUKMGUGDZQK6YHYA5Z6AY2G4XDSZPSZ3SW5UN3ARVMO6QSRDWP5YLEX",
		SourceAssetType:   horizonclient.AssetType4,
		DestinationAssets: "EURT:GAP5LETOV6YIE62YAM56STDANPRDO7ZFDBGSNHJQIYGGKSMOZAHOOS2S",
	}
	paths, err := client.StrictSendPaths(pr)
```

- Add `client.OfferDetails` ([#2303](https://github.com/stellar/go/pull/2303)).

```go
	client := horizonclient.DefaultPublicNetClient
	offer, err := client.OfferDetails("2")
	if err != nil {
		fmt.Println(err)
		return
	}	
	fmt.Print(offer)
```

- Add support to `client.Offers` for the filters: `Seller`, `Selling` and `Buying` ([#2230](https://github.com/stellar/go/pull/2230)).
```go
	offerRequest = horizonclient.OfferRequest{
		Seller:  "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
		Selling: "COP:GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
		Buying:  "EUR:GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
		Order:   horizonclient.OrderDesc,
	}	
	offers, err = client.Offers(offerRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(offers)
```
- Add `client.Accounts` ([#2229](https://github.com/stellar/go/pull/2229)).

This feature allows account retrieval filtering by signer or by a trustline to an asset.

```go
	client := horizonclient.DefaultPublicNetClient
	accountID := "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU"
	accountsRequest := horizonclient.AccountsRequest{Signer: accountID}	
	account, err := client.Accounts(accountsRequest)
	if err != nil {
		fmt.Println(err)
		return
	}	
	fmt.Print(account)
```

- Add `IsNotFoundError` ([#2197](https://github.com/stellar/go/pull/2197)).

### Deprecate

- Make `hProtocol.FeeStats` backwards compatible with Horizon `0.24.1` and `1.0` deprecating usage of `*_accepted_fee` ([#2290](https://github.com/stellar/go/pull/2290)).

All the `_accepted_fee` fields were removed in Horizon 1.0, however we extended this version of the SDK to backfill the `FeeStat` struct using data from `MaxFee`. This is a temporary workaround and it will be removed in horizonclient 3.0. Please start using data from `FeeStat.MaxFee` instead.


## [v2.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v2.0.0) - 2020-01-13

- Add custom `UnmarshalJSON()` implementations to Horizon protocol structs so `int64` fields can be parsed as JSON numbers or JSON strings
- Remove deprecated `fee_paid field` from Transaction response
- Dropped support for Go 1.10, 1.11.

## [v1.4.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.4.0) - 2019-08-09

- Add support for querying operation endpoint with `join` parameter [#1521](https://github.com/stellar/go/issues/1521).
- Add support for querying previous and next trade aggregations with `Client.NextTradeAggregationsPage` and `Client.PrevTradeAggregationsPage` methods.


## [v1.3.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.3.0) - 2019-07-08

- Transaction information returned by methods now contain new fields: `FeeCharged` and `MaxFee`. `FeePaid` is deprecated and will be removed in later versions.
- Improved unit test for `Client.FetchTimebounds` method.
- Added `Client.HomeDomainForAccount` helper method for retrieving the home domain of an account.

## [v1.2.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.2.0) - 2019-05-16

- Added support for returning the previous and next set of pages for a horizon response; issue [#985](https://github.com/stellar/go/issues/985).
- Fixed bug reported in [#1254](https://github.com/stellar/go/issues/1254)  that causes a panic when using horizonclient in goroutines.


## [v1.1.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.1.0) - 2019-05-02

### Added

- `Client.Root()` method for querying the root endpoint of a horizon server.
- Support for returning concrete effect types[#1217](https://github.com/stellar/go/pull/1217)
- Fix when no HTTP client is provided

### Changes

- `Client.Fund()` now returns `TransactionSuccess` instead of a http response pointer.

- Querying the effects endpoint now supports returning the concrete effect type for each effect. This is also supported in streaming mode. See the [docs](https://godoc.org/github.com/stellar/go/clients/horizonclient#Client.Effects) for examples.

## [v1.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.0) - 2019-04-26

 * Initial release
