package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// CreateAccount represents the Stellar create account operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type CreateAccount struct {
	destAccountID xdr.AccountId
	Destination   string
	Amount        string
	Asset         string // TODO: Not used yet
	xdrOp         xdr.CreateAccountOp
}

// Init for CreateAccount initialises the required XDR fields for this operation.
func (ca *CreateAccount) Init() error {
	err := ca.destAccountID.SetAddress(ca.Destination)
	if err != nil {
		return errors.Wrap(err, "Failed to set destination address")
	}
	ca.xdrOp.Destination = ca.destAccountID

	ca.xdrOp.StartingBalance, err = amount.Parse(ca.Amount)
	if err != nil {
		return errors.Wrap(err, "Failed to parse amount")
	}

	return err
}

// NewXDROperationBody for CreateAccount initialises the corresponding XDR body.
func (ca *CreateAccount) NewXDROperationBody() (xdr.OperationBody, error) {
	opType := xdr.OperationTypeCreateAccount
	return xdr.NewOperationBody(opType, ca.xdrOp)
}
