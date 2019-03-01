package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// BumpSequence represents the Stellar bump sequence operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type BumpSequence struct {
	BumpTo int64
	xdrOp  xdr.BumpSequenceOp
}

// BuildXDR for BumpSequence returns a fully configured XDR Operation.
func (bs *BumpSequence) BuildXDR() (xdr.Operation, error) {
	bs.xdrOp.BumpTo = xdr.SequenceNumber(bs.BumpTo)

	opType := xdr.OperationTypeBumpSequence
	body, err := xdr.NewOperationBody(opType, bs.xdrOp)

	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
