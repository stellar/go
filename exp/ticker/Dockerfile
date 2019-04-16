FROM golang:1.12-stretch

LABEL maintainer="Alex Cordeiro <alexc@stellar.org>"

EXPOSE 5432
EXPOSE 8000

RUN wget -q https://www.postgresql.org/media/keys/ACCC4CF8.asc -O- | apt-key add -
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ stretch-pgdg main" | tee /etc/apt/sources.list.d/postgresql.list

RUN apt-get update
RUN apt-get install -y \
    curl \
    git \
    libpq-dev \
    postgresql-client-11 \
    postgresql-11 \
    sudo \
    vim \
    supervisor \
    nginx

RUN useradd --uid 10011001 --home-dir /home/stellar --no-log-init stellar \
    && mkdir -p /home/stellar \
    && chown -R stellar:stellar /home/stellar

RUN mkdir -p /opt/stellar/bin
# TODO: get ticker binary directly from GitHub releases
COPY ticker /opt/stellar/bin/ticker
RUN mkdir -p /opt/stellar/www
RUN chown -R stellar:stellar /opt/stellar/www
RUN mkdir -p /opt/stellar/postgresql/data

COPY docker/conf /opt/stellar/conf

COPY start /
RUN ["chmod", "+x", "/start"]
ENTRYPOINT ["/start"]
