package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Inflation ...
type Inflation struct {
	xdrOp struct{}
}

// Build ...
func (inf *Inflation) Build() (xdr.Operation, error) {
	// Create operation body
	body, err := inf.NewXDROperationBody()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to create XDR")
	}
	// Append relevant operation to TX.operations
	xdrOperation := xdr.Operation{Body: body}

	return xdrOperation, nil
}

// NewXDROperationBody ...
func (inf *Inflation) NewXDROperationBody() (xdr.OperationBody, error) {
	// TODO: Better name
	// TODO: Add next two lines in here

	opType := xdr.OperationTypeInflation
	body, err := xdr.NewOperationBody(opType, nil)

	return body, err
}
