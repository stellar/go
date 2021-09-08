---
title: Frontier Quickstart
---

## Frontier Quickstart
This document describes how to quickly set up a **test** DigitalBits Core + Frontier node, that you can play around with to get a feel for how a digitalbits node operates. **This configuration is not secure!** It is **not** intended as a guide for production administration.

For detailed information about running Frontier and DigitalBits Core safely in production see the [Frontier Administration Guide](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/admin.md) and the [DigitalBits Core Administration Guide](https://github.com/xdbfoundation/DigitalBits/blob/master/docs/software/admin.md).

If you're ready to roll up your sleeves and dig into the code, check out the [Developer Guide](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/developing.md).

### Install and run the Quickstart Docker Image
The fastest way to get up and running is using the [DigitalBits Quickstart Docker Image](https://github.com/digitalbits/docker-digitalbits-core-frontier). This is a Docker container that provides both `digitalbits-core` and `frontier`, pre-configured for testing.

1. Install [Docker](https://www.docker.com/get-started).
2. Verify your Docker installation works: `docker run hello-world`
3. Create a local directory that the container can use to record state. This is helpful because it can take a few minutes to sync a new `digitalbits-core` with enough data for testing, and because it allows you to inspect and modify the configuration if needed. Here, we create a directory called `digitalbits` to use as the persistent volume:
`cd $HOME; mkdir digitalbits`
4. Download and run the DigitalBits Quickstart container, replacing `USER` with your username:

```bash
docker run --rm -it -p "8000:8000" -p "11626:11626" -p "11625:11625" -p"8002:5432" -v $HOME/digitalbits:/opt/digitalbits --name digitalbits digitalbits/quickstart --testnet
```

You can check out DigitalBits Core status by browsing to http://localhost:11626.

You can check out your Frontier instance by browsing to http://localhost:8000.

You can tail logs within the container to see what's going on behind the scenes:
```bash
docker exec -it digitalbits /bin/bash
supervisorctl tail -f digitalbits-core
supervisorctl tail -f frontier stderr
```

On a modern laptop this test setup takes about 15 minutes to synchronise with the last couple of days of testnet ledgers. At that point Frontier will be available for querying. 

See the [Quickstart Docker Image](https://github.com/digitalbits/docker-digitalbits-core-frontier) documentation for more details, and alternative ways to run the container. 

You can test your Frontier instance with a query like: http://localhost:8000/transactions?cursor=&limit=10&order=asc. Use the [DigitalBits Laboratory](https://laboratory.livenet.digitalbits.io) to craft other queries to try out,
and read about the available endpoints and see examples in the [Frontier API reference](https://github.com/xdbfoundation/go/blob/master/services/frontier/internal/docs/readme.md).

