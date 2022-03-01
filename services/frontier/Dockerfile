FROM ubuntu:focal

ARG FRONTIER_VERSION 

ENV DATABASE_URL="postgresql://user:password@localhost:5432/frontier"
ENV DIGITALBITS_CORE_DATABASE_URL="postgresql://user:password@localhost:5432/node1"
ENV DIGITALBITS_CORE_URL="http://deu-1.testnet.digitalbits.io:11626"
ENV HISTORY_ARCHIVE_URLS="https://history.testnet.digitalbits.io/node1"
ENV INGEST=false
ENV LOG_FILE="/var/log/digitalbits-frontier.log"
ENV NETWORK_PASSPHRASE="TestNet Global DigitalBits Network ; December 2020"
ENV FEE_PASSPHRASE="Testnet DigitalBits Fee Pool ; February 2021"
ENV PORT=80

RUN apt-get update && apt-get install curl postgresql-client libpq-dev -y
RUN curl -o frontier.deb https://archive.digitalbits.io/public/digitalbits-frontier/deb/ubuntu/pool/focal/main/d/di/digitalbits-frontier_${FRONTIER_VERSION}/digitalbits-frontier_${FRONTIER_VERSION}_amd64.deb \
&& dpkg -i frontier.deb && rm frontier.deb

ENTRYPOINT frontier db migrate up && frontier