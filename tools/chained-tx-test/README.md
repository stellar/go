# chained-tx-test

Chained-tx-test is a CLI utility that will create an account, then have that account submit a series of transactions one after the other with 100ms between each transaction submission.

The utility submits transactions to Horizon if no Core URL is provided, otherwise transactions are submitted directly to Core.

The utility prints out the ledger and duration of each transaction.

## Run against testnet submitting to Horizon

```
go run .
```

## Run against a local standalone instance submitting to Horizon

```
docker run --rm -it -p 8000:8000 --name stellar stellar/quickstart:latest --standalone
```

```
go run . -horizon=http://localhost:8000
```

## Run against a local standalone instance submitting to Core

```
docker run --rm -it -p 8000:8000 -p 11626:11626 --name stellar stellar/quickstart:latest --standalone
```

```
go run . -horizon=http://localhost:8000 -core=http://localhost:11626
```
