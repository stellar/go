FROM golang:1.12-stretch

LABEL maintainer="Alex Cordeiro <alexc@stellar.org>"

EXPOSE 5432
EXPOSE 8000

RUN apt-get update
RUN apt-get install -y \
    curl \
    git \
    libpq-dev \
    postgresql-client \
    postgresql \
    postgresql-contrib \
    sudo \
    vim \
    supervisor \
    nginx

RUN apt-get clean
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

RUN git clone https://github.com/stellar/go $GOPATH/src/stellar/go
WORKDIR $GOPATH/src/stellar/go
RUN dep ensure -v

WORKDIR $GOPATH/src/stellar/go/exp/ticker
RUN go build .
RUN mkdir -p /opt/stellar/bin
RUN cp ticker /opt/stellar/bin/ticker
RUN mkdir -p /opt/stellar/www
RUN mkdir -p /opt/stellar/postgresql/data


COPY docker/conf /opt/stellar/conf

# Add a user for accessing the database
RUN useradd --uid 10011001 --home-dir /home/stellar --no-log-init stellar \
    && mkdir -p /home/stellar \
    && chown -R stellar:stellar /home/stellar


COPY start /
RUN ["chmod", "+x", "/start"]
ENTRYPOINT ["/start"]
