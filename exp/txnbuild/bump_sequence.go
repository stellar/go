package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// BumpSequence represents the Stellar bump sequence operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type BumpSequence struct {
	BumpTo int64
}

// BuildXDR for BumpSequence returns a fully configured XDR Operation.
func (bs *BumpSequence) BuildXDR() (xdr.Operation, error) {
	var xdrOp xdr.BumpSequenceOp

	xdrOp.BumpTo = xdr.SequenceNumber(bs.BumpTo)

	opType := xdr.OperationTypeBumpSequence
	body, err := xdr.NewOperationBody(opType, xdrOp)

	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
