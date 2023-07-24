# Horizon
[![Build Status](https://circleci.com/gh/stellar/go.svg?style=shield)](https://circleci.com/gh/stellar/go)

Horizon is the client facing API server for the [Stellar ecosystem](https://developers.stellar.org/docs/start/introduction/).  It acts as the interface between [Stellar Core](https://developers.stellar.org/docs/run-core-node/) and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.

Check out the following resources to get started:
- [Horizon Development Guide](internal/docs/GUIDE_FOR_DEVELOPERS.md): Instructions for building and developing Horizon. Covers setup, building, testing, and contributing. Also contains some helpful notes and context for Horizon developers.
- [Quickstart Guide](internal/docs/QUICKSTART_GUIDE.md): A guide on setting up your own test Stellar Core + Horizon node using the quickstart Docker image.
- [Horizon Testing Guide](internal/docs/TESTING_NOTES.md): Details on how to test Horizon, including unit tests, integration tests, and end-to-end tests.
- [Horizon SDK and API Guide](internal/docs/SDK_API_GUIDE.md): Documentation on the Horizon SDKs, APIs, resources, and examples. Useful for developers building on top of Horizon.

## Run a production server
If you're an administrator planning to run a production instance of Horizon as part of the public Stellar network, you should check out the instructions on our public developer docs - [Run an API Server](https://developers.stellar.org/docs/run-api-server/). It covers installation, monitoring, error scenarios and more.

## Contributing
As an open source project, development of Horizon is public, and you can help! We welcome new issue reports, documentation and bug fixes, and contributions that further the project roadmap. The [Development Guide](internal/docs/GUIDE_FOR_DEVELOPERS.md) will show you how to build Horizon, see what's going on behind the scenes, and set up an effective develop-test-push cycle so that you can get your work incorporated quickly.
