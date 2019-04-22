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
	Selling Asset
	Buying  Asset
	Amount  string
	Price   string // TODO: Extend to include number, and n/d fraction. See package 'amount'
}

// BuildXDR for CreatePassiveOffer returns a fully configured XDR Operation.
func (cpo *CreatePassiveOffer) BuildXDR() (xdr.Operation, error) {
	xdrSelling, err := cpo.Selling.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set XDR 'Selling' field")
	}

	xdrBuying, err := cpo.Buying.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set XDR 'Buying' field")
	}

	xdrAmount, err := amount.Parse(cpo.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse 'Amount'")
	}

	xdrPrice, err := price.Parse(cpo.Price)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse 'Price'")
	}

	xdrOp := xdr.CreatePassiveOfferOp{
		Selling: xdrSelling,
		Buying:  xdrBuying,
		Amount:  xdrAmount,
		Price:   xdrPrice,
	}

	opType := xdr.OperationTypeCreatePassiveOffer
	body, err := xdr.NewOperationBody(opType, xdrOp)

	return xdr.Operation{Body: body}, errors.Wrap(err, "failed to build XDR OperationBody")
}
