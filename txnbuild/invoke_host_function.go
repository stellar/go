package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type InvokeHostFunctions struct {
	Functions     []xdr.HostFunction
	SourceAccount string
	Ext           xdr.TransactionExt
}

func (f *InvokeHostFunctions) BuildXDR() (xdr.Operation, error) {

	opType := xdr.OperationTypeInvokeHostFunction
	xdrOp := xdr.InvokeHostFunctionOp{
		Functions: f.Functions,
	}

	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR Operation")
	}

	op := xdr.Operation{Body: body}

	SetOpSourceAccount(&op, f.SourceAccount)
	return op, nil
}

func (f *InvokeHostFunctions) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetInvokeHostFunctionOp()
	if !ok {
		return errors.New("error parsing invoke host function operation from xdr")
	}

	f.SourceAccount = accountFromXDR(xdrOp.SourceAccount)
	f.Functions = result.Functions

	return nil
}

func (f *InvokeHostFunctions) Validate() error {
	if f.SourceAccount != "" {
		_, err := xdr.AddressToMuxedAccount(f.SourceAccount)
		if err != nil {
			return NewValidationError("SourceAccount", err.Error())
		}
	}
	return nil
}

func (f *InvokeHostFunctions) GetSourceAccount() string {
	return f.SourceAccount
}

func (f *InvokeHostFunctions) BuildTransactionExt() (xdr.TransactionExt, error) {
	return f.Ext, nil
}
