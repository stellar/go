# webauth

This is a [SEP-10] Web Authentication implementation based on SEP-10 v1.2.0
that requires the master key have a high threshold for authentication to
succeed.

SEP-10 defines an endpoint for authenticating a user in possession of a Stellar
account using their Stellar account as credentials. This implementation is a
standalone microservice that implements the minimum requirements as defined by
the SEP-10 protocol and will be adapted as the protocol evolves.

This implementation is not polished and is still experimental.
Running this implementation in production is not recommended.

## Usage

```
$ webauth --help
SEP-10 Web Authentication Server

Usage:
  webauth [command] [flags]
  webauth [command]

Available Commands:
  genjwtkey   Generate a JWT ECDSA key
  serve       Run the SEP-10 Web Authentication server

Flags:
  -h, --help   help for webauth

Use "webauth [command] --help" for more information about a command.
```

## Usage: Serve

```
$ webauth serve --help
Run the SEP-10 Web Authentication server

Usage:
  webauth serve [flags]

Flags:
      --challenge-expires-in int    The time period in seconds after which the challenge transaction expires (CHALLENGE_EXPIRES_IN) (default 300)
      --horizon-url string          Horizon URL used for looking up account details (HORIZON_URL) (default "https://horizon-testnet.stellar.org/")
      --jwt-expires-in int          The time period in seconds after which the JWT expires (JWT_EXPIRES_IN) (default 300)
      --jwt-key string              Base64 encoded ECDSA private key used for signing JWTs (JWT_KEY)
      --network-passphrase string   Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                    Port to listen and serve on (PORT) (default 8000)
      --signing-key string          Stellar signing key used for signing transactions (SIGNING_KEY)
```

[SEP-10]: https://github.com/stellar/stellar-protocol/blob/2be91ce8d8032ca9b2f368800d06b9fba346a147/ecosystem/sep-0010.md
