package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/price"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ManageBuyOffer represents the Stellar manage buy offer operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type ManageBuyOffer struct {
	Selling       Asset
	Buying        Asset
	Amount        string
	Price         string
	OfferID       int64
	SourceAccount Account
}

// BuildXDR for ManageBuyOffer returns a fully configured XDR Operation.
func (mo *ManageBuyOffer) BuildXDR() (xdr.Operation, error) {
	xdrSelling, err := mo.Selling.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set XDR 'Selling' field")
	}

	xdrBuying, err := mo.Buying.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set XDR 'Buying' field")
	}

	xdrAmount, err := amount.Parse(mo.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse 'Amount'")
	}

	xdrPrice, err := price.Parse(mo.Price)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse 'Price'")
	}

	opType := xdr.OperationTypeManageBuyOffer
	xdrOp := xdr.ManageBuyOfferOp{
		Selling:   xdrSelling,
		Buying:    xdrBuying,
		BuyAmount: xdrAmount,
		Price:     xdrPrice,
		OfferId:   xdr.Int64(mo.OfferID),
	}
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR OperationBody")
	}

	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, mo.SourceAccount)
	return op, nil
}

// FromXDR for ManageBuyOffer initialises the txnbuild struct from the corresponding xdr Operation.
func (mo *ManageBuyOffer) FromXDR(xdrOp xdr.Operation) error {
	result, ok := xdrOp.Body.GetManageBuyOfferOp()
	if !ok {
		return errors.New("error parsing manage_buy_offer operation from xdr")
	}

	if xdrOp.SourceAccount != nil {
		mo.SourceAccount = &SimpleAccount{AccountID: xdrOp.SourceAccount.Address()}
	}

	mo.OfferID = int64(result.OfferId)
	mo.Amount = amount.String(result.BuyAmount)
	mo.Price = price.StringFromFloat64(float64(result.Price.N / result.Price.D))

	buyingAsset, err := assetFromXDR(result.Buying)
	if err != nil {
		return errors.Wrap(err, "error parsing manage_buy_offer operation from xdr")
	}
	mo.Buying = buyingAsset

	sellingAsset, err := assetFromXDR(result.Selling)
	if err != nil {
		return errors.Wrap(err, "error parsing manage_buy_offer operation from xdr")
	}
	mo.Selling = sellingAsset
	return nil
}
