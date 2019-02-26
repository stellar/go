package txnbuild

import "github.com/stellar/go/network"

// StellarNetwork is a global setting that sets the choice of network to use.
var StellarNetwork = network.TestNetworkPassphrase

// UseTestNetwork sets the global network setting to use the Stellar TestNet.
func UseTestNetwork() {
	StellarNetwork = network.TestNetworkPassphrase
}

// UsePublicNetwork sets the global network setting to use the Stellar Public network
// (i.e. production).
func UsePublicNetwork() {
	StellarNetwork = network.PublicNetworkPassphrase
}
