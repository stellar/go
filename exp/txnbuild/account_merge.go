package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// AccountMerge represents the Stellar merge account operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type AccountMerge struct {
	destAccountID xdr.AccountId
	Destination   string
	xdrOp         xdr.AccountId
}

// BuildXDR for AccountMerge returns a fully configured XDR Operation.
func (am *AccountMerge) BuildXDR() (xdr.Operation, error) {
	err := am.destAccountID.SetAddress(am.Destination)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set destination address")
	}
	am.xdrOp = am.destAccountID

	opType := xdr.OperationTypeAccountMerge
	body, err := xdr.NewOperationBody(opType, am.xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
