#! /usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
GOTOP="$( cd "$DIR/../../../../../../../.." && pwd )"
PACKAGES=$(find $GOTOP/src/github.com/stellar/go/services/horizon/internal/test/scenarios -iname '*.rb' -not -name '_common_accounts.rb')
# PACKAGES=$(find $GOTOP/src/github.com/stellar/go/services/horizon/internal/test/scenarios -iname 'kahuna.rb')

go install github.com/stellar/go/services/horizon

dropdb hayashi_scenarios --if-exists
createdb hayashi_scenarios

export STELLAR_CORE_DATABASE_URL="postgres://localhost/hayashi_scenarios?sslmode=disable"
export DATABASE_URL="postgres://localhost/horizon_scenarios?sslmode=disable"
export NETWORK_PASSPHRASE="Test SDF Network ; September 2015"
export STELLAR_CORE_URL="http://localhost:8080"
export SKIP_CURSOR_UPDATE="true"

# run all scenarios
for i in $PACKAGES; do
  CORE_SQL="${i%.rb}-core.sql"
  HORIZON_SQL="${i%.rb}-horizon.sql"
  bundle exec scc -r $i --dump-root-db > $CORE_SQL

  # load the core scenario
  psql $STELLAR_CORE_DATABASE_URL < $CORE_SQL

  # recreate horizon dbs
  dropdb horizon_scenarios --if-exists
  createdb horizon_scenarios

  # import the core data into horizon
  $GOTOP/bin/horizon db init
  $GOTOP/bin/horizon db rebase

  # write horizon data to sql file
  pg_dump $DATABASE_URL \
    --clean --if-exists --no-owner --no-acl --inserts \
    | sed '/SET idle_in_transaction_session_timeout/d' \
    | sed '/SET row_security/d' \
    > $HORIZON_SQL
done


# commit new sql files to bindata
go generate github.com/stellar/go/services/horizon/internal/test/scenarios
# go test github.com/stellar/go/services/horizon/internal/ingest
