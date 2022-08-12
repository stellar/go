FROM golang:1.18 AS builder

WORKDIR /go/src/github.com/stellar/go
COPY . ./
RUN go mod download
RUN go install github.com/stellar/go/exp/lighthorizon/index/cmd/batch/map
RUN go install github.com/stellar/go/exp/lighthorizon/index/cmd/batch/reduce

FROM ubuntu:20.04
ENV DEBIAN_FRONTEND=noninteractive
# ca-certificates are required to make tls connections
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl wget gnupg apt-utils
RUN apt-get clean

COPY --from=builder /go/src/github.com/stellar/go/exp/lighthorizon/build/index-batch/start ./
COPY --from=builder /go/bin/map ./
COPY --from=builder /go/bin/reduce ./
RUN ["chmod", "+x", "/start"]

ENTRYPOINT ["/start"]
