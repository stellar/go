#! /usr/bin/env bash

set -e

SCENARIO=$1
CORE_SQL=$GOPATH/src/github.com/stellar/go/services/horizon/test/scenarios/$SCENARIO-core.sql
HORIZON_SQL=$GOPATH/src/github.com/stellar/go/services/horizon/test/scenarios/$SCENARIO-horizon.sql

echo "psql $STELLAR_CORE_DATABASE_URL < $CORE_SQL" 
psql $STELLAR_CORE_DATABASE_URL < $CORE_SQL 
echo "psql $DATABASE_URL < $HORIZON_SQL"
psql $DATABASE_URL < $HORIZON_SQL 
