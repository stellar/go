# Clients package

Packages here provide client libraries for accessing the ecosystem of DigitalBits services.

* `frontierclient` - programmatic client access to Frontier (use in conjunction with [txnbuild](../txnbuild))
* `digitalbitstoml` - parse DigitalBits.toml files from the internet
* `federation` - resolve federation addresses into digitalbits account IDs, suitable for use within a transaction
* `frontier` (DEPRECATED) - the original Frontier client, now superceded by `frontierclient`

See [GoDoc](https://godoc.org/github.com/xdbfoundation/go/clients) for more details.

## For developers: Adding new client packages

Ideally, each one of our client packages will have commonalities in their API to ease the cost of learning each.  It's recommended that we follow a pattern similar to the `net/http` package's client shape:

A type, `Client`, is the central type of any client package, and its methods should provide the bulk of the functionality for the package.  A `DefaultClient` var is provided for consumers that don't need client-level customization of behavior.  Each method on the `Client` type should have a corresponding func at the package level that proxies a call through to the default client.  For example, `http.Get()` is the equivalent of `http.DefaultClient.Get()`.
