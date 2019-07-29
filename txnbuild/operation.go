package txnbuild

import (
	"github.com/stellar/go/xdr"
)

// Operation represents the operation types of the Stellar network.
type Operation interface {
	BuildXDR() (xdr.Operation, error)
}

// SetOpSourceAccount sets the source account ID on an Operation.
func SetOpSourceAccount(op *xdr.Operation, sourceAccount Account) {
	if sourceAccount == nil {
		return
	}
	var opSourceAccountID xdr.AccountId
	opSourceAccountID.SetAddress(sourceAccount.GetAccountID())
	op.SourceAccount = &opSourceAccountID
}

func operationFromXDR(xdrOp xdr.Operation) (Operation, error) {
	var newOp Operation
	var err error
	switch xdrOp.Body.Type {
	case xdr.OperationTypeCreateAccount:
		var op CreateAccount
		err = op.FromXDR(xdrOp)
		newOp = &op
	case xdr.OperationTypePayment:
		var op Payment
		err = op.FromXDR(xdrOp)
		newOp = &op
	}

	if err != nil {
		return nil, err
	}

	return newOp, nil
}
