# Run keystored in development

Generate the certificate and the private key pair for localhost
if you haven't done so:

```sh
cd github.com/stellar/go/exp/services/keystore
./tls/regen.sh
```
Simply choose all the default options. This will create three files:
tls/server.crt, tls/server.key, and tls/server.csr.

We will only be using `server.crt` and `server.key`.

## Install the `keystored` command:

```sh
cd github.com/stellar/go/exp/services/keystore
go install ./cmd/keystored
```

## Set up `keystore` Postgres database locally:

```sh
createdb keystore
keystored migrate up
```

You can undo all migrations by running
```sh
keystored migrate down
```

You can redo the last migration by running
```sh
keystored migrate redo
```

You can check whether there is any unapplied migrations by running
```sh
keystored migrate status
```

## Run `keystored` in development with authentication disabled:

You might want to set the `KEYSTORE_LISTENER_PORT` environment variable
for the keystored listener. Otherwise, the default value is port 8000.

```sh
keystored -tls-cert=tls/server.crt -tls-key=tls/server.key -auth=false serve
```

Before you have a valid endpoint that can handle your auth token and return a
user id in plaintext, you might want to disable authentication for testing.

## Run `keystored` in production with authentication disabled:

There are four environment variables used for starting keystored:
`KEYSTORE_DATABASE_URL`, `DB_MAX_IDLE_CONNS`, `DB_MAX_OPEN_CONNS`, `KEYSTORE_AUTHFORWARDING_URL`.
* `KEYSTORE_DATABASE_URL` and `KEYSTORE_AUTHFORWARDING_URL` are required.
* `DB_MAX_IDLE_CONNS` and `DB_MAX_OPEN_CONNS` are default to 5.
* `KEYSTORE_AUTHFORWARDING_URL` is ignored when authentication is turned off.

```sh
keystored -tls-cert=PATH_TO_TLS_CERT -tls-key=PATH_TO_TLS_KEY -auth=false serve
```

## Logging

You can put the log messages in a designated file with the `-log-file` flag as well as determine
the log severity level with the `-log-level` flag:

```sh
keystored -log-file=PATH_TO_YOUR_LOG_FILE -log-level=[debug|info|warn|error] serve
```
