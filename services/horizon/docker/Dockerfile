FROM golang:1.12-alpine3.10 AS builder

RUN apk --no-cache add curl git mercurial
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/github.com/stellar/go
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . ./
RUN go install github.com/stellar/go/services/horizon

FROM alpine:3.10
# ca-certificates are required to make tls connections
RUN apk add --no-cache ca-certificates

COPY --from=builder /go/bin/horizon ./
COPY ./services/horizon/docker/horizon-entry.sh ./
ENTRYPOINT ["./horizon-entry.sh"]