#! /bin/bash
set -e

cd "$(dirname "${BASH_SOURCE[0]}")"

export HORIZON_INTEGRATION_TESTS=true
export HORIZON_INTEGRATION_ENABLE_CAP_35=true
export HORIZON_INTEGRATION_ENABLE_CAPTIVE_CORE=false
export CAPTIVE_CORE_BIN=/usr/bin/stellar-core
export TRACY_NO_INVARIANT_CHECK=1

if [[ "$(docker inspect integration_postgres -f '{{.State.Running}}')" != "true" ]]; then
  docker rm -f integration_postgres || true;
  docker run -d --name integration_postgres --env POSTGRES_HOST_AUTH_METHOD=trust -p 5432:5432 circleci/postgres:9.6.5-alpine
fi

exec go test -timeout 25m github.com/stellar/go/services/horizon/internal/integration/... "$@"
