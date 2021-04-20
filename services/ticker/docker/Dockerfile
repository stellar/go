FROM golang:1.16.3 as build

ADD . /src/ticker
WORKDIR /src/ticker
RUN go build -o /bin/ticker ./services/ticker


FROM ubuntu:18.04

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates
COPY --from=build /bin/ticker /app/
EXPOSE 8000
ENTRYPOINT ["/app/ticker"]
