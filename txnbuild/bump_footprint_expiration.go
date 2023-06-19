package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type BumpFootprintExpiration struct {
	LedgersToExpire uint32
	SourceAccount   string
}

func (f *BumpFootprintExpiration) BuildXDR() (xdr.Operation, error) {
	xdrOp := xdr.BumpFootprintExpirationOp{
		Type:            xdr.BumpFootprintExpirationTypeBumpFootprintExpirationUniform,
		LedgersToExpire: new(xdr.Uint32),
	}
	*xdrOp.LedgersToExpire = xdr.Uint32(f.LedgersToExpire)

	body, err := xdr.NewOperationBody(xdr.OperationTypeBumpFootprintExpiration, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR Operation")
	}

	op := xdr.Operation{Body: body}

	SetOpSourceAccount(&op, f.SourceAccount)
	return op, nil
}

func (f *BumpFootprintExpiration) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetBumpFootprintExpirationOp()
	if !ok {
		return errors.New("error parsing invoke host function operation from xdr")
	}
	f.SourceAccount = accountFromXDR(xdrOp.SourceAccount)
	f.LedgersToExpire = uint32(*result.LedgersToExpire)
	return nil
}

func (f *BumpFootprintExpiration) Validate() error {
	if f.SourceAccount != "" {
		_, err := xdr.AddressToMuxedAccount(f.SourceAccount)
		if err != nil {
			return NewValidationError("SourceAccount", err.Error())
		}
	}
	return nil
}

func (f *BumpFootprintExpiration) GetSourceAccount() string {
	return f.SourceAccount
}
