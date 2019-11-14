#! /bin/bash
set -e

/etc/init.d/postgresql start

while ! psql -U circleci -d core -h localhost -p 5432 -c 'select 1' >/dev/null 2>&1; do
    echo "Waiting for postgres to be available..."
    sleep 1
done

echo "using version $(stellar-core version)"

if [ -z ${TESTNET+x} ]; then
    stellar-core --conf ./stellar-core.cfg new-db
    export LATEST_LEDGER=`curl -v --max-time 10 --retry 3 --retry-connrefused --retry-delay 5 https://history.stellar.org/prd/core-live/core_live_001/.well-known/stellar-history.json | jq -r '.currentLedger'`
else
    stellar-core --conf ./stellar-core-testnet.cfg new-db
    export LATEST_LEDGER=`curl -v --max-time 10 --retry 3 --retry-connrefused --retry-delay 5 https://history.stellar.org/prd/core-testnet/core_testnet_001/.well-known/stellar-history.json | jq -r '.currentLedger'`
fi
echo "Latest ledger: $LATEST_LEDGER"

if ! ./run_test.sh; then
    curl -X POST --data-urlencode "payload={ \"username\": \"ingestion-check\", \"text\": \"ingestion dump (git commit \`$GITCOMMIT\`) of ledger \`$LATEST_LEDGER\` does not match stellar core db.\"}" $SLACK_URL
    exit 1
fi