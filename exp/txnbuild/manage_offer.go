package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/price"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

//CreateOfferOp returns a ManageSellOffer operation to create a new offer, by
// setting the OfferID to "0".
func CreateOfferOp(selling, buying Asset, amount, price string) ManageSellOffer {
	return ManageSellOffer{
		Selling: selling,
		Buying:  buying,
		Amount:  amount,
		Price:   price,
		OfferID: 0,
	}
}

//UpdateOfferOp returns a ManageSellOffer operation to update an offer.
func UpdateOfferOp(selling, buying Asset, amount, price string, offerID int64) ManageSellOffer {
	return ManageSellOffer{
		Selling: selling,
		Buying:  buying,
		Amount:  amount,
		Price:   price,
		OfferID: offerID,
	}
}

//DeleteOfferOp returns a ManageSellOffer operation to delete an offer, by
// setting the Amount to "0".
func DeleteOfferOp(offerID int64) ManageSellOffer {
	// It turns out Stellar core doesn't care about any of these fields except the amount.
	// However, Horizon will reject ManageSellOffer if it is missing fields.
	// Horizon will also reject if Buying == Selling.
	// Therefore unfortunately we have to make up some dummy values here.
	return ManageSellOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{Code: "FAKE", Issuer: "GBAQPADEYSKYMYXTMASBUIS5JI3LMOAWSTM2CHGDBJ3QDDPNCSO3DVAA"},
		Amount:  "0",
		Price:   "1",
		OfferID: offerID,
	}
}

// ManageSellOffer represents the Stellar manage offer operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type ManageSellOffer struct {
	Selling Asset
	Buying  Asset
	Amount  string
	Price   string // TODO: Extend to include number, and n/d fraction. See package 'amount'
	OfferID int64
}

// BuildXDR for ManageSellOffer returns a fully configured XDR Operation.
func (mo *ManageSellOffer) BuildXDR() (xdr.Operation, error) {
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

	opType := xdr.OperationTypeManageSellOffer
	xdrOp := xdr.ManageSellOfferOp{
		Selling: xdrSelling,
		Buying:  xdrBuying,
		Amount:  xdrAmount,
		Price:   xdrPrice,
		OfferId: xdr.Int64(mo.OfferID),
	}
	body, err := xdr.NewOperationBody(opType, xdrOp)

	return xdr.Operation{Body: body}, errors.Wrap(err, "failed to build XDR OperationBody")
}
