package txnbuild

import "github.com/stellar/go/network"

// UseTestNetwork sets the global network setting to use the Stellar TestNet.
func UseTestNetwork() string {
	return network.TestNetworkPassphrase
}

// UsePublicNetwork sets the global network setting to use the Stellar Public network
// (i.e. production).
func UsePublicNetwork() string {
	return network.PublicNetworkPassphrase
}
