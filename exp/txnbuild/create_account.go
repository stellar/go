package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// CreateAccount represents the Stellar create account operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type CreateAccount struct {
	Destination string
	Amount      string
}

// BuildXDR for CreateAccount returns a fully configured XDR Operation.
func (ca *CreateAccount) BuildXDR() (xdr.Operation, error) {
	var xdrOp xdr.CreateAccountOp

	err := xdrOp.Destination.SetAddress(ca.Destination)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set destination address")
	}

	xdrOp.StartingBalance, err = amount.Parse(ca.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to parse amount")
	}

	opType := xdr.OperationTypeCreateAccount
	body, err := xdr.NewOperationBody(opType, xdrOp)

	return xdr.Operation{Body: body}, errors.Wrap(err, "Failed to build XDR OperationBody")
}
