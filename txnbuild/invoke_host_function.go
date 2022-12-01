package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type InvokeHostFunction struct {
	Function      xdr.HostFunction
	Footprint     xdr.LedgerFootprint
	SourceAccount string
}

func (f *InvokeHostFunction) BuildXDR() (xdr.Operation, error) {

	opType := xdr.OperationTypeInvokeHostFunction
	xdrOp := xdr.InvokeHostFunctionOp{
		Function:  f.Function,
		Footprint: f.Footprint,
	}

	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR Operation")
	}

	op := xdr.Operation{Body: body}

	SetOpSourceAccount(&op, f.SourceAccount)
	return op, nil
}

func (f *InvokeHostFunction) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetInvokeHostFunctionOp()
	if !ok {
		return errors.New("error parsing invoke host function operation from xdr")
	}

	f.SourceAccount = accountFromXDR(xdrOp.SourceAccount)
	f.Footprint = result.Footprint
	f.Function = result.Function

	return nil
}

func (f *InvokeHostFunction) Validate() error {
	if f.SourceAccount != "" {
		_, err := xdr.AddressToMuxedAccount(f.SourceAccount)
		if err != nil {
			return NewValidationError("SourceAccount", err.Error())
		}
	}
	return nil
}

func (f *InvokeHostFunction) GetSourceAccount() string {
	return f.SourceAccount
}
