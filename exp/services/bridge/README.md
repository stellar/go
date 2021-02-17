# Bridge

Bridge is a lightweight API server that manages virtual Stellar accounts that
are federated behind a domain, and exposes a simple interface for a
higher-level service to call to send payments from the virtual accounts, and
monitor the receipt of payments to them.

## Usage

### Create DB
```
psql -c 'CREATE DATABASE bridge;' || true
psql -d bridge -f ./exp/services/bridge/db.sql
```

### Run Bridge

```
export DOMAIN=localhost
export ACCOUNT=G...
exoprt PORT=8080
go run ./exp/services/bridge
```

### Call Bridge

#### Create a Virtual Account

```
http POST localhost:8080/create-virtual-account username=user1
```

#### Lookup Federated Address

```
http GET localhost:8080/federation\?type=name\&q=user\*localhost
```

#### Send Payment to Stellar

TODO

#### Get Payments from Stellar

TODO
