# Running DigitalBits with Docker Compose

## Dependencies

The only dependency you will need to install is [Docker](https://www.docker.com/products/docker-desktop).

## Start script

[start.sh](./start.sh) will setup the env file and run docker-compose to start the DigitalBits docker containers. Feel free to use this script, otherwise continue with the next two steps.

The script takes one optional parameter which configures the DigitalBits network used by the docker containers. If no parameter is supplied, the containers will run on the DigitalBits test network.

`./start.sh pubnet` will run the containers on the DigitalBits public network.

`./start.sh standalone` will run the containers on a private standalone DigitalBits network.

## Set up a .env file

Mac OS X and Windows users should create an [`.env`](https://docs.docker.com/compose/environment-variables/#the-env_file-configuration-option) file which consists of:

`NETWORK_MODE=bridge`

Linux users should also create an `.env` file. However, the contents of the file should look like:

`NETWORK_MODE=host`

Additionally, you will need to add `127.0.0.1 host.docker.internal` to the `/etc/hosts` file on your linux machine.

If https://github.com/docker/for-linux/issues/264 is ever fixed then it won't be necessary to alias `host.docker.internal` to localhost and there won't be any differences between the Linux and Mac OS X / Windows configurations.


## Run docker-compose

Run the following command to start all the DigitalBits docker containers:

```
docker-compose up -d --build
```

Frontier will be exposed on port 8000. DigitalBits Core will be exposed on port 11626. The DigitalBits Core postgres instance will be exposed on port 5641.
The Frontier postgres instance will be exposed on port 5432.

## Swapping in a local service

If you're developing a service locally you may want to run that service locally while also being able to interact with the other DigitalBits components running in Docker. You can do that by stopping the container corresponding to the service you're developing.

For example, to run Frontier locally from source, you would perform the following steps:

```
# stop frontier in docker-compose
docker-compose stop frontier
```

Now you can run frontier locally in vscode using the following configuration:
```
    {
        "name": "Launch",
        "type": "go",
        "request": "launch",
        "mode": "debug",
        "remotePath": "",
        "port": 2345,
        "host": "127.0.0.1",
        "program": "${workspaceRoot}/services/frontier/main.go",
        "env": {
            "DATABASE_URL": "postgres://postgres@localhost:5432/frontier?sslmode=disable",
            "DIGITALBITS_CORE_DATABASE_URL": "postgres://postgres:mysecretpassword@localhost:5641/digitalbits?sslmode=disable",
            "NETWORK_PASSPHRASE": "TestNet Global DigitalBits Network ; December 2020",
            "DIGITALBITS_CORE_URL": "http://localhost:11626",
            "INGEST": "true",
        },
        "args": []
    }
```

Similarly, to run DigitalBits core locally from source and have it interact with Frontier in docker, all you need to do is run `docker-compose stop core` before running DigitalBits core from source.

## Connecting to the DigitalBits Public Network

By default, the Docker Compose file configures DigitalBits Core to connect to the DigitalBits test network. If you would like to run the docker containers on the
DigitalBits public network, run `docker-compose -f docker-compose.yml -f docker-compose.pubnet.yml up -d --build`. 

To run the containers on a private stand-alone network, run `docker-compose -f docker-compose.yml -f docker-compose.standalone.yml up -d --build`.
When you run DigitalBits Core on a private stand-alone network, an account will be created which will hold 100 billion DigitalBits.
The seed for the account will be emitted in the DigitalBits Core logs:

```
2020-04-22T18:39:19.248 GD5KD [Ledger INFO] Root account seed: SC5O7VZUXDJ6JBDSZ74DSERXL7W3Y5LTOAMRF7RQRL3TAGAPS7LUVG3L
```

When running Frontier on a private stand-alone network, Frontier will not start ingesting until DigitalBits Core creates its first history archive snapshot. DigitalBits Core creates snapshots every 64 ledgers, which means ingestion will be delayed until ledger 64.

When you switch between different networks you will need to clear the DigitalBits Core and DigitalBits Frontier databases. You can wipe out the databases by running `docker-compose down --remove-orphans -v`.

## Using a specific version of DigitalBits Core

By default the Docker Compose file is configured to use the latest version of DigitalBits Core. To use a specific version, you can edit [docker-compose.yml](./docker-compose.yml) and set the appropriate [tag](https://hub.docker.com/r/digitalbits/digitalbits-core/tags) on the DigitalBits Core docker image
