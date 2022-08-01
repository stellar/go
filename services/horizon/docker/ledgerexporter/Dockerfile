FROM golang:1.18 AS builder

ARG GIT_COMMIT
WORKDIR /
RUN git clone https://github.com/stellar/go.git stellar-go
WORKDIR /stellar-go

# By default "git fetch" only fetches refs/<branchname>
# Below ensures we also fetch PR refs
RUN git config --add remote.origin.fetch "+refs/pull/*/head:refs/remotes/origin/pull/*"
RUN git fetch --force --quiet origin
RUN if [ -n "${GIT_COMMIT}" ]; then git checkout "${GIT_COMMIT}"; fi
RUN go build -v ./exp/services/ledgerexporter

FROM ubuntu:20.04
ARG STELLAR_CORE_VERSION
ENV STELLAR_CORE_VERSION=${STELLAR_CORE_VERSION:-*}
ENV STELLAR_CORE_BINARY_PATH /usr/bin/stellar-core

ENV DEBIAN_FRONTEND=noninteractive
# ca-certificates are required to make tls connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl wget gnupg apt-utils
RUN wget -qO - https://apt.stellar.org/SDF.asc | APT_KEY_DONT_WARN_ON_DANGEROUS_USAGE=true apt-key add -
RUN echo "deb https://apt.stellar.org focal stable" >/etc/apt/sources.list.d/SDF.list
RUN echo "deb https://apt.stellar.org focal unstable" >/etc/apt/sources.list.d/SDF-unstable.list
RUN apt-get update && apt-get install -y stellar-core=${STELLAR_CORE_VERSION}
RUN apt-get clean

ADD captive-core-pubnet.cfg /
ADD captive-core-testnet.cfg /

ADD start /
RUN ["chmod", "+x", "start"]

# for the captive core sqlite database
RUN mkdir -p /cc

COPY --from=builder /stellar-go/ledgerexporter ./

ENTRYPOINT ["/start"]
