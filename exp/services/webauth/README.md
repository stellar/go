# webauth

This is a SEP-10 Web Authentication implementation based on SEP-10 v1.2.0.

This implementation is not polished and is still experimental. Running this implementation in production is not recommended.

## Usage

````
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
      --challenge-expires-in int    The time period after which the challenge transaction expires (default 300000000000)
      --horizon-url string          Horizon URL used for looking up account details (default "https://horizon-testnet.stellar.org/")
      --jwt-expires-in int          The time period that after which the JWT expires (default 18000000000000)
      --jwt-key string              Base64 encoded ECDSA private key used for signing JWTs
      --network-passphrase string   Network passphrase of the Stellar network transactions should be signed for (default "Test SDF Network ; September 2015")
      --port int                    Port to listen and serve on (default 8000)
      --signing-key string          Stellar signing key used for signing transactions
```
