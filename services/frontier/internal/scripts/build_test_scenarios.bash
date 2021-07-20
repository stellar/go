#! /usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
GOTOP="$( cd "$DIR/../../../../../../../.." && pwd )"
PACKAGES=$(find $GOTOP/src/github.com/xdbfoundation/go/services/frontier/internal/test/scenarios -iname '*.rb' -not -name '_common_accounts.rb')
#PACKAGES=$(find $GOTOP/src/github.com/xdbfoundation/go/services/frontier/internal/test/scenarios -iname 'failed_transactions.rb')

go install github.com/xdbfoundation/go/services/frontier

dropdb hayashi_scenarios --if-exists
createdb hayashi_scenarios

export DIGITALBITS_CORE_DATABASE_URL="postgres://localhost/hayashi_scenarios?sslmode=disable"
export DATABASE_URL="postgres://localhost/frontier_scenarios?sslmode=disable"
export NETWORK_PASSPHRASE="TestNet Global DigitalBits Network ; December 2020"
export DIGITALBITS_CORE_URL="http://localhost:8080"
export SKIP_CURSOR_UPDATE="true"
export INGEST_FAILED_TRANSACTIONS=true

# run all scenarios
for i in $PACKAGES; do
  echo $i
  CORE_SQL="${i%.rb}-core.sql"
  FRONTIER_SQL="${i%.rb}-frontier.sql"
  scc -r $i --allow-failed-transactions --dump-root-db > $CORE_SQL

  # load the core scenario
  psql $DIGITALBITS_CORE_DATABASE_URL < $CORE_SQL

  # recreate frontier dbs
  dropdb frontier_scenarios --if-exists
  createdb frontier_scenarios

  # import the core data into frontier
  $GOTOP/bin/frontier db init
  $GOTOP/bin/frontier db init-asset-stats
  $GOTOP/bin/frontier db rebase

  # write frontier data to sql file
  pg_dump $DATABASE_URL \
    --clean --if-exists --no-owner --no-acl --inserts \
    | sed '/SET idle_in_transaction_session_timeout/d' \
    | sed '/SET row_security/d' \
    > $FRONTIER_SQL
done


# commit new sql files to bindata
go generate github.com/xdbfoundation/go/services/frontier/internal/test/scenarios
# go test github.com/xdbfoundation/go/services/frontier/internal/ingest
