package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/xdr"
)

func populateCreateAccountOperation(
	op *xdr.Operation,
	transactionEnvelope *xdr.TransactionEnvelope,
	baseOp operations.Base,
) (operations.CreateAccount, error) {
	createAccount := op.Body.CreateAccountOp
	baseOp.Type = "create_account"

	return operations.CreateAccount{
		Base:            baseOp,
		StartingBalance: amount.String(createAccount.StartingBalance),
		Funder:          sourceAccount(op, transactionEnvelope),
		Account:         createAccount.Destination.Address(),
	}, nil
}
