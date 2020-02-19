# Recovery Signer

This is a [SEP-XX] Recovery Signer implementation based on SEP-XX v?.?.?.

A Recovery Signer is a server that can help a user regain control of a Stellar
account if they have lost their secret key. A user registers their account with
a Recovery Signer by adding it as a signer, and informs the Recovery Signer
that any user proving access to a phone number or email address can have
transactions signed. A user who has registered their account with two or more
Recovery Signers can recover the account with their help.

This implementation is not polished and is still experimental.
Running this implementation in production is not recommended.

## Usage

```
$ recoverysigner --help
SEP-XX Recovery Signer server

Usage:
  recoverysigner [command] [flags]
  recoverysigner [command]

Available Commands:
  serve       Run the SEP-XX Recover Signer server

Use "recoverysigner [command] --help" for more information about a command.
```

## Usage: Serve

```
$ recoverysigner serve --help
Run the SEP-XX Recover Signer server

Usage:
  recoverysigner serve [flags]

Flags:
      --firebase-project-id string    Firebase project ID to use for validating Firebase JWTs (FIREBASE_PROJECT_ID)
      --horizon-url string            Horizon URL used for looking up account details (HORIZON_URL) (default "https://horizon-testnet.stellar.org/")
      --network-passphrase string     Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                      Port to listen and serve on (PORT) (default 8000)
      --sep10-jwt-public-key string   Base64 encoded ECDSA public key used to validate SEP-10 JWTs (SEP10_JWT_PUBLIC_KEY)
      --signing-key string            Stellar signing key used for signing transactions (will be deprecated with per-account keys in the future) (SIGNING_KEY)
```

[SEP-XX]: https://github.com/stellar/stellar-protocol/...
