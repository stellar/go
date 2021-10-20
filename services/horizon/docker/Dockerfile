FROM ubuntu:focal

ARG VERSION=2.9.0-165
ARG CORE_VERSION=18.0.3-746.f3baea6.focal
ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y wget apt-transport-https gnupg2 && \
    wget -qO /etc/apt/trusted.gpg.d/SDF.asc https://apt.stellar.org/SDF.asc && \
    echo "deb https://apt.stellar.org focal stable" | tee -a /etc/apt/sources.list.d/SDF.list && \
    apt-get update && apt-cache madison stellar-horizon stellar-core && apt-get install -y stellar-horizon=${VERSION} stellar-core=${CORE_VERSION} && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /var/log/*.log /var/log/*/*.log

EXPOSE 8000
ENTRYPOINT ["/usr/bin/stellar-horizon"]
