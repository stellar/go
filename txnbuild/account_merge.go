package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// AccountMerge represents the Stellar merge account operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type AccountMerge struct {
	Destination   string
	SourceAccount string
}

// BuildXDR for AccountMerge returns a fully configured XDR Operation.
func (am *AccountMerge) BuildXDR(withSEP23 bool) (xdr.Operation, error) {
	var xdrOp xdr.MuxedAccount
	var err error
	if withSEP23 {
		err = xdrOp.SetAddressWithSEP23(am.Destination)
	} else {
		err = xdrOp.SetAddress(am.Destination)
	}
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set destination address")
	}

	opType := xdr.OperationTypeAccountMerge
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR OperationBody")
	}
	op := xdr.Operation{Body: body}
	if withSEP23 {
		SetOpSourceMuxedAccount(&op, am.SourceAccount)
	} else {
		SetOpSourceAccount(&op, am.SourceAccount)
	}
	return op, nil
}

// FromXDR for AccountMerge initialises the txnbuild struct from the corresponding xdr Operation.
func (am *AccountMerge) FromXDR(xdrOp xdr.Operation, withMuxedAccounts bool) error {
	if xdrOp.Body.Type != xdr.OperationTypeAccountMerge {
		return errors.New("error parsing account_merge operation from xdr")
	}

	am.SourceAccount = accountFromXDR(xdrOp.SourceAccount, withMuxedAccounts)
	if xdrOp.Body.Destination != nil {
		if withMuxedAccounts {
			am.Destination = xdrOp.Body.Destination.SEP23Address()
		} else {
			aid := xdrOp.Body.Destination.ToAccountId()
			am.Destination = aid.Address()
		}
	}

	return nil
}

// Validate for AccountMerge validates the required struct fields. It returns an error if any of the fields are
// invalid. Otherwise, it returns nil.
func (am *AccountMerge) Validate(withMuxedAccounts bool) error {
	var err error
	if withMuxedAccounts {
		_, err = xdr.AddressToAccountId(am.Destination)
	} else {
		_, err = xdr.SEP23AddressToMuxedAccount(am.Destination)
	}
	if err != nil {
		return NewValidationError("Destination", err.Error())
	}
	return nil
}

// GetSourceAccount returns the source account of the operation, or the empty string if not
// set.
func (am *AccountMerge) GetSourceAccount() string {
	return am.SourceAccount
}
