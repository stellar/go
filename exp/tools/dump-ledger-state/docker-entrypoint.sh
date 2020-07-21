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
else
    stellar-core --conf ./stellar-core-testnet.cfg new-db
fi

if [ -z ${LATEST_LEDGER+x} ]; then
    # Get latest ledger
    echo "Getting latest checkpoint ledger..."
    if [ -z ${TESTNET+x} ]; then
        export LATEST_LEDGER=`curl -s http://history.stellar.org/prd/core-live/core_live_001/.well-known/stellar-history.json | jq -r '.currentLedger'`
    else
        export LATEST_LEDGER=`curl -s http://history.stellar.org/prd/core-testnet/core_testnet_001/.well-known/stellar-history.json | jq -r '.currentLedger'`
    fi
fi

if [[ -z "${LATEST_LEDGER}" ]]; then
  echo "could not obtain latest ledger"
  exit 1
fi

echo "Latest ledger: $LATEST_LEDGER"

if ! ./run_test.sh; then
    curl -X POST --data-urlencode "payload={ \"username\": \"ingestion-check\", \"text\": \"@horizon-team ingestion dump (git commit \`$GITCOMMIT\`) of ledger \`$LATEST_LEDGER\` does not match stellar core db.\"}" $SLACK_URL
    exit 1
fi