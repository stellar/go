FROM golang:1.16 as build

ADD . /src/regulated-assets-approval-server-demo
WORKDIR /src/regulated-assets-approval-server-demo
RUN go build -o /bin/regulated-assets-approval-server-demo ./services/regulated-assets-approval-server-demo


FROM ubuntu:20.04

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates
COPY --from=build /bin/regulated-assets-approval-server-demo /app/
EXPOSE 8000
ENTRYPOINT ["/app/regulated-assets-approval-server-demo"]
CMD ["serve"]
