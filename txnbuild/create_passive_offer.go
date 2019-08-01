package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/price"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// CreatePassiveSellOffer represents the Stellar create passive offer operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type CreatePassiveSellOffer struct {
	Selling       Asset
	Buying        Asset
	Amount        string
	Price         string
	SourceAccount Account
}

// BuildXDR for CreatePassiveSellOffer returns a fully configured XDR Operation.
func (cpo *CreatePassiveSellOffer) BuildXDR() (xdr.Operation, error) {
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

	xdrOp := xdr.CreatePassiveSellOfferOp{
		Selling: xdrSelling,
		Buying:  xdrBuying,
		Amount:  xdrAmount,
		Price:   xdrPrice,
	}

	opType := xdr.OperationTypeCreatePassiveSellOffer
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR OperationBody")
	}
	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, cpo.SourceAccount)
	return op, nil
}

// FromXDR for CreatePassiveSellOffer initialises the txnbuild struct from the corresponding xdr Operation.
func (cpo *CreatePassiveSellOffer) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetCreatePassiveSellOfferOp()
	if !ok {
		return errors.New("error parsing create_passive_sell_offer operation from xdr")
	}

	if xdrOp.SourceAccount != nil {
		cpo.SourceAccount = &SimpleAccount{AccountID: xdrOp.SourceAccount.Address()}
	}

	cpo.Amount = amount.String(result.Amount)
	cpo.Price = price.StringFromFloat64(float64(result.Price.N / result.Price.D))

	buyingAsset, err := assetFromXDR(result.Buying)
	if err != nil {
		return errors.Wrap(err, "error parsing create_passive_sell_offer operation from xdr")
	}
	cpo.Buying = buyingAsset

	sellingAsset, err := assetFromXDR(result.Selling)
	if err != nil {
		return errors.Wrap(err, "error parsing create_passive_sell_offer operation from xdr")
	}
	cpo.Selling = sellingAsset
	return nil
}
