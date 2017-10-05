#! /usr/bin/env bash
set -e

# This scripts rebuilds the latest.sql file included in the schema package.

go generate github.com/stellar/go/services/horizon/db2/schema
go install github.com/stellar/go/services/horizon/cmd/horizon
dropdb horizon_schema --if-exists
createdb horizon_schema
DATABASE_URL=postgres://localhost/horizon_schema?sslmode=disable $GOPATH/bin/horizon db migrate up

DUMP_OPTS="--schema=public --no-owner --no-acl --inserts"
LATEST_PATH="$GOPATH/src/github.com/stellar/go/services/horizon/db2/schema/latest.sql"
BLANK_PATH="$GOPATH/src/github.com/stellar/go/services/horizon/test/scenarios/blank-horizon.sql"

pg_dump postgres://localhost/horizon_schema?sslmode=disable $DUMP_OPTS \
  | sed '/SET idle_in_transaction_session_timeout/d'  \
  | sed '/SET row_security/d' \
  > $LATEST_PATH
pg_dump postgres://localhost/horizon_schema?sslmode=disable \
  --clean --if-exists $DUMP_OPTS \
  | sed '/SET idle_in_transaction_session_timeout/d'  \
  | sed '/SET row_security/d' \
  > $BLANK_PATH

go generate github.com/stellar/go/services/horizon/db2/schema
go generate github.com/stellar/go/services/horizon/test
go install github.com/stellar/go/services/horizon/cmd/horizon
