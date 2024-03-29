FROM golang:1.20 AS builder

WORKDIR /go/src/github.com/stellar/go
COPY . ./
RUN go mod download
RUN go install github.com/stellar/go/exp/services/ledgerexporter

FROM ubuntu:22.04
ARG STELLAR_CORE_VERSION
ENV STELLAR_CORE_VERSION=${STELLAR_CORE_VERSION:-*}
ENV STELLAR_CORE_BINARY_PATH /usr/bin/stellar-core

ENV DEBIAN_FRONTEND=noninteractive
# ca-certificates are required to make tls connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl wget gnupg apt-utils
RUN wget -qO - https://apt.stellar.org/SDF.asc | APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE=true apt-key add -
RUN echo "deb https://apt.stellar.org jammy stable" >/etc/apt/sources.list.d/SDF.list
RUN echo "deb https://apt.stellar.org jammy unstable" >/etc/apt/sources.list.d/SDF-unstable.list
RUN apt-get update && apt-get install -y stellar-core=${STELLAR_CORE_VERSION}
RUN apt-get clean

COPY --from=builder /go/src/github.com/stellar/go/exp/lighthorizon/build/ledgerexporter/captive-core-pubnet.cfg /
COPY --from=builder /go/src/github.com/stellar/go/exp/lighthorizon/build/ledgerexporter/captive-core-testnet.cfg /
COPY --from=builder /go/src/github.com/stellar/go/exp/lighthorizon/build/ledgerexporter/start /

RUN ["chmod", "+x", "/start"]

# for the captive core sqlite database
RUN mkdir -p /cc

COPY --from=builder /go/bin/ledgerexporter ./

ENTRYPOINT ["/start"]
