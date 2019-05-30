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

Install the `keystored` command:

```sh
cd github.com/stellar/go/exp/services/keystore
go install ./cmd/keystored
```

Set up `keystore` Postgres database locally:

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

Run `keystored` in development:

```sh
keystored -tls-cert=tls/server.crt -tls-key=tls/server.key serve
```

