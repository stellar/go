FROM golang:1.14-stretch AS builder

WORKDIR /go/src/github.com/stellar/go
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go install github.com/stellar/go/services/horizon

FROM ubuntu:18.04

ENV STELLAR_CORE_VERSION 14.1.1-1355-5b2989d3
ENV STELLAR_CORE_BINARY_PATH /usr/local/bin/stellar-core

# ca-certificates are required to make tls connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget


RUN apt-get install -y --no-install-recommends libpqxx-4.0v5 curl
RUN wget -O stellar-core.deb https://s3.amazonaws.com/stellar.org/releases/stellar-core/stellar-core-${STELLAR_CORE_VERSION}_amd64.deb
RUN dpkg -i stellar-core.deb
RUN rm stellar-core.deb

RUN apt-get clean

COPY --from=builder /go/bin/horizon ./

ENTRYPOINT ["./horizon"]
