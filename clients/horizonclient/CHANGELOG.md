# Changelog

All notable changes to this project will be documented in this
file.  This project adheres to [Semantic Versioning](http://semver.org/).


## [v1.1.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.1.0) - 2019-02-02

### Added

- `Client.Root()` method for querying the root endpoint of a horizon server.
- Support for returning concrete effect types[#1217](https://github.com/stellar/go/pull/1217)
- Fix when no HTTP client is provided

### Changes

- `Client.Fund()` now returns `TransactionSuccess` instead of a http response pointer.


## [v1.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.0) - 2019-04-26

 * Initial release