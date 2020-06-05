# webauth

This is a [SEP-10] Web Authentication implementation based on SEP-10 v1.3.0
that requires a user to prove they possess a signing key(s) that meets the high
threshold for an account, i.e. they have the ability to perform any high
threshold operation on the given account. If an account does not exist it may
be optionally verified using the account's master key.

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
  genjwk      Generate a JSON Web Key (ECDSA/ES256) for JWT issuing
  serve       Run the SEP-10 Web Authentication server

Use "webauth [command] --help" for more information about a command.
```

## Usage: Serve

```
$ webauth serve --help
Run the SEP-10 Web Authentication server

Usage:
  webauth serve [flags]

Flags:
      --allow-accounts-that-do-not-exist   Allow accounts that do not exist (ALLOW_ACCOUNTS_THAT_DO_NOT_EXIST)
      --challenge-expires-in int           The time period in seconds after which the challenge transaction expires (CHALLENGE_EXPIRES_IN) (default 300)
      --horizon-url string                 Horizon URL used for looking up account details (HORIZON_URL) (default "https://horizon-testnet.stellar.org/")
      --jwk string                         JSON Web Key (JWK) used for signing JWTs (if the key is an asymmetric key that has separate public and private key, the JWK must contain the private key) (JWK)
      --jwt-expires-in int                 The time period in seconds after which the JWT expires (JWT_EXPIRES_IN) (default 300)
      --jwt-issuer string                  The issuer to set in the JWT iss claim (JWT_ISSUER)
      --network-passphrase string          Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                           Port to listen and serve on (PORT) (default 8000)
      --signing-key string                 Stellar signing key(s) used for signing transactions comma separated (first key is used for signing, others used for verifying challenges) (SIGNING_KEY)
```

[SEP-10]: https://github.com/stellar/stellar-protocol/blob/2be91ce8d8032ca9b2f368800d06b9fba346a147/ecosystem/sep-0010.md
