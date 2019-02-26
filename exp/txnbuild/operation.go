package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Operation represents the operation types of the Stellar network.
type Operation interface {
	Init() error
	NewXDROperationBody() (xdr.OperationBody, error)
}

// BuildOperation for Operation fully configures the Operation.
func BuildOperation(op Operation) (xdr.Operation, error) {
	err := op.Init()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to initialise Operation")
	}
	body, err := op.NewXDROperationBody()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR Operation")
	}

	return xdr.Operation{Body: body}, nil
}
