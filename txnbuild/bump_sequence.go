package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// BumpSequence represents the Stellar bump sequence operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type BumpSequence struct {
	BumpTo        int64
	SourceAccount Account
}

// BuildXDR for BumpSequence returns a fully configured XDR Operation.
func (bs *BumpSequence) BuildXDR() (xdr.Operation, error) {
	opType := xdr.OperationTypeBumpSequence
	xdrOp := xdr.BumpSequenceOp{BumpTo: xdr.SequenceNumber(bs.BumpTo)}
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR OperationBody")
	}
	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, bs.SourceAccount)
	return op, nil
}
