# Stellar Go 
[![Build Status](https://circleci.com/gh/stellar/go.svg?style=shield)](https://circleci.com/gh/stellar/go)
[![GoDoc](https://godoc.org/github.com/stellar/go?status.svg)](https://godoc.org/github.com/stellar/go)
[![Go Report Card](https://goreportcard.com/badge/github.com/stellar/go)](https://goreportcard.com/report/github.com/stellar/go)

This repo is the home for all of the public go code produced by SDF.  In addition to various tools and services, this repository is the SDK from which you may develop your own applications that integrate with the stellar network.

## Package Index

* [Horizon Server](services/horizon): Full-featured API server for Stellar network
* [Go Horizon SDK - horizonclient](clients/horizonclient): Client for Horizon server (queries and transaction submission)
* [Go Horizon SDK - txnbuild](txnbuild): Construct Stellar transactions and operations
* [Bifrost](services/bifrost): Bitcoin/Ethereum -> Stellar bridge
* [Ticker](services/ticker): An API server that provides statistics about assets and markets on the Stellar network
* [Keystore](services/keystore): An API server that is used to store and manage encrypted keys for Stellar client applications
* Servers for Anchors & Financial Institutions
  * [Bridge Server](services/bridge): send payments and take action when payments are received
  * [Compliance Server](services/compliance): Allows financial institutions to exchange KYC information
  * [Federation Server](services/federation): Allows organizations to provide addresses for users (`jane*examplebank.com`)

## Dependencies

This repository is officially supported on the last two releases of Go, which is currently Go 1.12 and Go 1.13.

It depends on a [number of external dependencies](./go.mod), and uses Go [Modules](https://github.com/golang/go/wiki/Modules) to manage them. Running any `go` command will automatically download dependencies required for that operation.

You can choose to checkout this repository into a [GOPATH](https://github.com/golang/go/wiki/GOPATH) or into any directory, but if you are using a GOPATH with Go 1.12 or earlier you must set environment variable `GO111MODULE=on` to enable Modules.

## Directory Layout

In addition to the other top-level packages, there are a few special directories that contain specific types of packages:

* **clients** contains packages that provide client packages to the various Stellar services.
* **exp** contains experimental packages.  Use at your own risk.
* **handlers** contains packages that provide pluggable implementors of `http.Handler` that make it easier to incorporate portions of the Stellar protocol into your own http server. 
* **support** contains packages that are not intended for consumption outside of Stellar's other packages.  Packages that provide common infrastructure for use in our services and tools should go here, such as `db` or `log`. 
* **support/scripts** contains single-file go programs and bash scripts used to support the development of this repo. 
* **services** contains packages that compile to applications that are long-running processes (such as API servers).
* **tools** contains packages that compile to command line applications.

Each of these directories have their own README file that explain further the nature of their contents.

### Other packages

In addition to the packages described above, this repository contains various packages related to working with the Stellar network from a go program.  It's recommended that you use [godoc](https://godoc.org/github.com/stellar/go#pkg-subdirectories) to browse the documentation for each.


## Package source layout

While much of the code in individual packages is organized based upon different developers' personal preferences, many of the packages follow a simple convention for organizing the declarations inside of a package that aim to aid in your ability to find code.

In each package, there may be one or more of a set of common files:

- *main.go*: Every package should have a `main.go` file.  This file contains the package documentation (unless a separate `doc.go` file is used), _all_ of the exported vars, consts, types and funcs for the package. 
- *internal.go*:  This file should contain unexported vars, consts, types, and funcs.  Conceptually, it should be considered the private counterpart to the `main.go` file of a package
- *errors.go*: This file should contains declarations (both types and vars) for errors that are used by the package.
- *example_test.go*: This file should contains example tests, as described at https://blog.golang.org/examples.

In addition to the above files, a package often has files that contains code that is specific to one declared type.  This file uses the snake case form of the type name (for example `loggly_hook.go` would correspond to the type `LogglyHook`).  This file should contain method declarations, interface implementation assertions and any other declarations that are tied solely to that type.

Each non-test file can have a test counterpart like normal, whose name ends with `_test.go`.  The common files described above also have their own test counterparts... for example `internal_test.go` should contains tests that test unexported behavior and more commonly test helpers that are unexported.

Generally, file contents are sorted by exported/unexported, then declaration type  (ordered as consts, vars, types, then funcs), then finally alphabetically.

### Test helpers

Often, we provide test packages that aid in the creation of tests that interact with our other packages.  For example, the `support/db` package has the `support/db/dbtest` package underneath it that contains elements that make it easier to test code that accesses a SQL database.  We've found that this pattern of having a separate test package maximizes flexibility and simplifies package dependencies.


## Coding conventions

- Always document exported package elements: vars, consts, funcs, types, etc.
- Tests are better than no tests.
