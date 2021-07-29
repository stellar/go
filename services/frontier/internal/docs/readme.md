---
title: Frontier
---
Frontier is the server for the client facing API for the DigitalBits ecosystem.  It acts as the interface between [digitalbits-core](https://developers.digitalbits.io/software/#digitalbits-core) and applications that want to access the DigitalBits network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams, etc. See [an overview of the DigitalBits ecosystem](https://developers.digitalbits.io/guides/get-started/index.html) for more details.

You can interact directly with frontier via curl or a web browser but XDB provides a [JavaScript SDK](https://developers.digitalbits.io/xdb-digitalbits-sdk/reference/index.html) for clients to use to interact with Frontier.

XDB Foundation runs an instance of Frontier that is connected to the testnet [https://frontier.testnet.digitalbits.io/](https://frontier.testnet.digitalbits.io/) and an instance of Frontier that is connected to the livenet [https://frontier.livenet.digitalbits.io/](https://frontier.livenet.digitalbits.io/)

## Libraries

XDB Foundation maintained libraries:

- [JavaScript](https://github.com/xdbfoundation/xdb-digitalbits-sdk)
- [Go](https://github.com/xdbfoundation/go/tree/master/clients/frontierclient)