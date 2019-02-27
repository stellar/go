package txnbuild

import "github.com/stellar/go/xdr"

// BumpSequence represents the Stellar bump sequence operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type BumpSequence struct {
	BumpTo int64
	xdrOp  xdr.BumpSequenceOp
}

// Init for BumpSequence initialises the required XDR fields for this operation.
func (bs *BumpSequence) Init() error {
	bs.xdrOp.BumpTo = xdr.SequenceNumber(bs.BumpTo)

	return nil
}

// NewXDROperationBody for BumpSequence initialises the corresponding XDR body.
func (bs *BumpSequence) NewXDROperationBody() (xdr.OperationBody, error) {
	opType := xdr.OperationTypeBumpSequence
	return xdr.NewOperationBody(opType, bs.xdrOp)
}
