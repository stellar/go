#!/usr/bin/env bash

set -e
set -x

source /etc/profile
# work within the current docker working dir
if [ ! -f "./stellar-core.cfg" ]; then
   cp /stellar-core.cfg ./
fi   

echo "using config:"
cat stellar-core.cfg

if [ "$1" = "standalone" ]; then
  # initialize for new history archive path, remove any pre-existing on same path from base image
  if [ -d "/bootstrap/history" ]; then
    ls /bootstrap/
    rm -rf ./stellar.db ./buckets
    cp /bootstrap/stellar.db .
    cp -r /bootstrap/buckets ./buckets
    mkdir -p ./history
    rm -rf ./history/vs
    cp -r /bootstrap/history/vs ./history/vs
  else
    # initialize new db
    stellar-core new-db
    rm -rf ./history
    stellar-core new-hist vs
  fi

  # serve history archives to horizon on port 1570
  pushd ./history/vs/
  python3 -m http.server 1570 &
  popd
else
  # initialize new db
  stellar-core new-db
fi

exec stellar-core run --console
