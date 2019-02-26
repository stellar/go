package txnbuild

import (
	"github.com/stellar/go/xdr"
)

// Inflation represents the Stellar inflation operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type Inflation struct {
	xdrOp struct{}
}

// Init for Inflation is a no-op to match the txnbuild.Operation interface.
func (inf *Inflation) Init() error {
	return nil
}

// NewXDROperationBody for Inflation initialises the corresponding XDR body.
func (inf *Inflation) NewXDROperationBody() (xdr.OperationBody, error) {
	opType := xdr.OperationTypeInflation
	return xdr.NewOperationBody(opType, nil)
}
