package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Inflation represents the Stellar inflation operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type Inflation struct {
	xdrOp struct{}
}

// BuildXDR for Inflation returns a fully configured XDR Operation.
func (inf *Inflation) BuildXDR() (xdr.Operation, error) {
	opType := xdr.OperationTypeInflation
	body, err := xdr.NewOperationBody(opType, nil)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
