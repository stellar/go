package txnbuild

import (
	"github.com/stellar/go/xdr"
)

// Operation represents the operation types of the Stellar network.
type Operation interface {
	BuildXDR() (xdr.Operation, error)
}
