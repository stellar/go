# bifrost

## Config

* `port` - bifrost server listening port
* `using_proxy` (default `false`) - set to `true` if bifrost lives behind a proxy or load balancer
* `bitcoin`
  * `master_public_key` - master public key for bitcoin keys derivation (read more in [BIP-0032](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki))
  * `rpc_server` - URL of [bitcoin-core](https://github.com/bitcoin/bitcoin) >= 0.15.0 RPC server
  * `rpc_user` (default empty) - username for RPC server (if any)
  * `rpc_pass` (default empty) - password for RPC server (if any)
  * `testnet` (default `false`) - set to `true` if you're testing bifrost in ethereum
  * `minimum_value_btc` - minimum transaction value in BTC that will be accepted by Bifrost, everything below will be ignored.
* `ethereum`
  * `master_public_key` - master public key for bitcoin keys derivation (read more in [BIP-0032](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki))
  * `rpc_server` - URL of [geth](https://github.com/ethereum/go-ethereum) >= 1.7.1 RPC server
  * `network_id` - network ID (`3` - Ropsten testnet, `1` - live Ethereum network)
  * `minimum_value_eth` - minimum transaction value in ETH that will be accepted by Bifrost, everything below will be ignored.
* `stellar`
  * `issuer_public_key` - public key of the assets issuer or hot wallet,
  * `signer_secret_key` - issuer's secret key if only one instance of Bifrost is deployed OR [channel](https://www.stellar.org/developers/guides/channels.html)'s secret key if more than one instance of Bifrost is deployed. Signer's sequence number will be consumed in transaction's sequence number.
  * `horizon` - URL to [horizon](https://github.com/stellar/go/tree/master/services/horizon) server
  * `network_passphrase` - Stellar network passphrase (`Public Global Stellar Network ; September 2015` for production network, `Test SDF Network ; September 2015` for test network)
* `database`
  * `type` - currently the only supported database type is: `postgres`
  * `dsn` - data source name for postgres connection (`postgres://user:password@host/dbname?sslmode=sslmode` - [more info](https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters))

## Going to production

* Remember than everyone with master public key and **any** child private key can recover your **master** private key. Do not share your master public key and obviously any private keys. Treat your master public key as if it was a private key. Read more in BIP-0032 [Security](https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki#security) section.
* Make sure "Sell [your token] for BTC" and/or "Sell [your token] for ETH" exist in Stellar production network.
* Make sure you don't use account from `stellar.issuer_secret_key` anywhere else than bifrost. Otherwise, sequence numbers will go out of sync and bifrost will stop working. It's good idea to create a new signer on issuing account.
* Check public master key correct. Use CLI tool to generate a few addresses and ensure you have corresponding private keys! You should probably send test transactions to these addresses and check if you can withdraw funds.
* Make sure `using_proxy` variable is set to correct value. Otherwise you will see your proxy IP instead of users' IPs in logs.
* Make sure you're not connecting to testnets.
* Deploy at least 2 bifrost, bitcoin-core, geth, stellar-core and horizon servers. Use multi-AZ database.
* Do not use SDF's horizon servers. There is no SLA and we cannot guarantee it will handle your load.
* Make sure bifrost <-> bitcoin-core and bifrost <-> geth connections are not public or are encrypted (mitm attacks).
* Make sure that "Authorization required" [flag](https://www.stellar.org/developers/guides/concepts/accounts.html#flags) is not set on your issuing account. It's a good idea to set "Authorization revocable" flag during ICO stage to remove trustlines to accounts with lost keys.
* Monitor bifrost logs and react to all WARN and ERROR entries.
* Make sure you are using geth >= 1.7.1 and bitcoin-core >= 0.15.0.
* Turn off horizon rate limiting.
* Make sure you configured minimum accepted value for Bitcoin and Ethereum transactions.
* Make sure you start from a fresh DB in production. If Bifrost was running, you stopped it and then started again all the block mined during that that will be processed what can take a lot of time.

## Stress-testing

//
