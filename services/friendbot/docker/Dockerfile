FROM golang:1.16.2 as build

ADD . /src/friendbot
WORKDIR /src/friendbot
RUN go build -o /bin/friendbot ./services/friendbot

FROM ubuntu:18.04

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates
COPY --from=build /bin/friendbot /app/
EXPOSE 8004
ENTRYPOINT ["/app/friendbot"]
