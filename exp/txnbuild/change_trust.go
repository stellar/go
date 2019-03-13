package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ChangeTrust represents the Stellar change trust operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type ChangeTrust struct {
	Line  *Asset
	Limit string
	xdrOp xdr.ChangeTrustOp
}

// NewRemoveTrustlineOp returns a ChangeTrust operation to remove the trustline of the described asset,
// by setting the limit to "0".
func NewRemoveTrustlineOp(issuedAsset *Asset) ChangeTrust {
	return ChangeTrust{
		Line:  issuedAsset,
		Limit: "0",
	}
}

// BuildXDR for ChangeTrust returns a fully configured XDR Operation.
func (ct *ChangeTrust) BuildXDR() (xdr.Operation, error) {
	var err error
	if ct.Line.IsNative() {
		return xdr.Operation{}, errors.New("Trustline cannot be extended to a native (XLM) asset")
	}
	ct.xdrOp.Line, err = ct.Line.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Can't convert asset for trustline to XDR")
	}

	ct.xdrOp.Limit, err = amount.Parse(ct.Limit)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to parse limit amount")
	}

	opType := xdr.OperationTypeChangeTrust
	body, err := xdr.NewOperationBody(opType, ct.xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
