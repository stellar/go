package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/price"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// CreatePassiveOffer represents the Stellar create passive offer operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type CreatePassiveOffer struct {
	Selling *Asset
	Buying  *Asset
	Amount  string
	Price   string // TODO: Extend to include number, and n/d fraction. See package 'amount'
}

// BuildXDR for CreatePassiveOffer returns a fully configured XDR Operation.
func (cpo *CreatePassiveOffer) BuildXDR() (xdr.Operation, error) {
	var err error
	var xdrOp xdr.CreatePassiveOfferOp

	xdrOp.Selling, err = cpo.Selling.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set XDR 'Selling' field")
	}

	xdrOp.Buying, err = cpo.Buying.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set XDR 'Buying' field")
	}

	xdrOp.Amount, err = amount.Parse(cpo.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to parse 'Amount'")
	}

	xdrOp.Price, err = price.Parse(cpo.Price)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to parse 'Price'")
	}

	opType := xdr.OperationTypeCreatePassiveOffer
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to build XDR OperationBody")
	}

	return xdr.Operation{Body: body}, nil
}
