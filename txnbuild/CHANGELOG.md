# Changelog

All notable changes to this project will be documented in this
file.  This project adheres to [Semantic Versioning](http://semver.org/).

## Unreleased [v1.2.0] 

* In addition to account responses from horizon, transactions and operations can now be built with txnbuild.SimpleAccount structs constructed locally. 
* Added `MaxTrustlineLimit` which represents the maximum value for a trustline limit.
* ChangeTrust operation with no `Limit` field set now defaults to `MaxTrustlineLimit`.

## [v1.1.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.1.0) - 2019-02-02

* Support for multiple signatures ([#1198](https://github.com/stellar/go/pull/1198))

## [v1.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.0) - 2019-04-26

* Initial release
