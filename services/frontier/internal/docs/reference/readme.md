---
title: Overview
---

Frontier is an API server for the DigitalBits ecosystem.  It acts as the interface between [digitalbits-core](https://github.com/xdbfoundation/DigitalBits) and applications that want to access the DigitalBits network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams, etc. See [an overview of the DigitalBits ecosystem](https://developers.digitalbits.io/guides/get-started/index.html) for details of where Frontier fits in.

Frontier provides a RESTful API to allow client applications to interact with the DigitalBits network. You can communicate with Frontier using cURL or just your web browser. However, if you're building a client application, you'll likely want to use a DigitalBits SDK in the language of your client.
XDB Foundation provides a [JavaScript SDK](https://developers.digitalbits.io/xdb-digitalbits-sdk/reference/index.html) for clients to use to interact with Frontier.

XDB Foundation runs an instance of Frontier that is connected to the DigitalBits testnet: [https://frontier.testnet.digitalbits.io/](frontier.testnet.digitalbits.io) and one that is connected to the DigitalBits livenet:
[https://frontier.livenet.digitalbits.io/](https://frontier.livenet.digitalbits.io/).

## Libraries

XDB Foundation maintained libraries:

- [JavaScript](https://github.com/xdbfoundation/xdb-digitalbits-sdk)
- [Go](https://github.com/xdbfoundation/go/tree/master/clients/frontierclient)

