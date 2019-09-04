## [UNRELEASED]
- Added nested `"issuer_detail"` field to `/assets.json`.
- Dropped support for Go 1.10, 1.11.


## [v1.1.0] - 2019-07-22

- Added support for running the ticker on Stellar's Test Network, by using the `--testnet` CLI flag.
- The ticker now retries requests to Horizon if it gets rate-limited.
- Minor bug fixes and performance improvements.


## [v1.0.0] - 2019-05-20

Initial release of the ticker.
