FROM golang:1.17 AS builder

WORKDIR /go/src/github.com/stellar/go
COPY . ./
RUN go mod download
RUN go install github.com/stellar/go/exp/lighthorizon/index/cmd/single

FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive
# ca-certificates are required to make tls connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl wget gnupg apt-utils
RUN apt-get clean

COPY --from=builder /go/bin/single ./

ENTRYPOINT ./single \
  -source "$TXMETA_SOURCE" \
  -target "$INDEXES_SOURCE" \
  -network-passphrase "$NETWORK_PASSPHRASE" \
  -start "$START" \
  -end "$END" \
  -modules "$MODULES" \
  -watch="$WATCH" \
  -workers "$WORKERS"
    
