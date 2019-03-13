package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ManageData represents the Stellar manage data operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type ManageData struct {
	Name  string
	Value []byte
	xdrOp xdr.ManageDataOp
}

// BuildXDR for ManageData returns a fully configured XDR Operation.
func (md *ManageData) BuildXDR() (xdr.Operation, error) {
	md.xdrOp.DataName = xdr.String64(md.Name)

	// No data value clears the named data entry on the account
	if md.Value == nil {
		md.xdrOp.DataValue = nil
	} else {
		xdrDV := xdr.DataValue(md.Value)
		md.xdrOp.DataValue = &xdrDV
	}

	opType := xdr.OperationTypeManageData
	body, err := xdr.NewOperationBody(opType, md.xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
