package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// CreateAccount ...
type CreateAccount struct {
	destAccountID xdr.AccountId
	Destination   string
	Amount        string
	Asset         string // TODO: Not used yet
	xdrOp         xdr.CreateAccountOp
}

// Build ...
func (ca *CreateAccount) Build() (xdr.Operation, error) {
	// Create operation body
	body, err := ca.NewXDROperationBody()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to create XDR")
	}
	// Append relevant operation to TX.operations
	xdrOperation := xdr.Operation{Body: body}

	return xdrOperation, nil
}

// NewXDROperationBody ...
func (ca *CreateAccount) NewXDROperationBody() (xdr.OperationBody, error) {
	// TODO: Better name
	// TODO: Add next two lines in here
	// TODO: Check both errors

	err := ca.Init()
	opType := xdr.OperationTypeCreateAccount
	body, err := xdr.NewOperationBody(opType, ca.xdrOp)

	return body, err
}

// Init ...
func (ca *CreateAccount) Init() error {
	err := ca.destAccountID.SetAddress(ca.Destination)
	ca.xdrOp.Destination = ca.destAccountID

	// TODO: Wrap error
	ca.xdrOp.StartingBalance, err = amount.Parse(ca.Amount)

	return err
}
