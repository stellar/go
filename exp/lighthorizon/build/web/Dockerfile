FROM golang:1.20 AS builder

WORKDIR /go/src/github.com/stellar/go
COPY . ./
RUN go mod download
RUN go install github.com/stellar/go/exp/lighthorizon

FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
# ca-certificates are required to make tls connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl wget gnupg apt-utils
RUN apt-get clean

COPY --from=builder /go/bin/lighthorizon ./

ENTRYPOINT ./lighthorizon serve \
  --network-passphrase "$NETWORK_PASSPHRASE" \
  --parallel-downloads "$MAX_PARALLEL_DOWNLOADS" \
  --ledger-cache "$CACHE_PATH" \
  --ledger-cache-preload "$CACHE_PRELOAD_COUNT" \
  --ledger-cache-preload-start "$CACHE_PRELOAD_START_LEDGER" \
  --log-level debug \
  "$TXMETA_SOURCE" "$INDEXES_SOURCE"
