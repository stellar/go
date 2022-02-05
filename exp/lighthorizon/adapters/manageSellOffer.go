package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func populateManageSellOfferOperation(
	op *xdr.Operation,
	baseOp operations.Base,
) (operations.ManageSellOffer, error) {
	manageSellOffer := op.Body.ManageSellOfferOp
	baseOp.Type = "manage_sell_offer"

	var (
		buyingAssetType string
		buyingCode      string
		buyingIssuer    string
	)
	err := manageSellOffer.Buying.Extract(&buyingAssetType, &buyingCode, &buyingIssuer)
	if err != nil {
		return operations.ManageSellOffer{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	var (
		sellingAssetType string
		sellingCode      string
		sellingIssuer    string
	)
	err = manageSellOffer.Selling.Extract(&sellingAssetType, &sellingCode, &sellingIssuer)
	if err != nil {
		return operations.ManageSellOffer{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	return operations.ManageSellOffer{
		Offer: operations.Offer{
			Base:   baseOp,
			Amount: amount.String(manageSellOffer.Amount),
			Price:  manageSellOffer.Price.String(),
			PriceR: base.Price{
				N: int32(manageSellOffer.Price.N),
				D: int32(manageSellOffer.Price.D),
			},
			BuyingAssetType:    buyingAssetType,
			BuyingAssetCode:    buyingCode,
			BuyingAssetIssuer:  buyingIssuer,
			SellingAssetType:   sellingAssetType,
			SellingAssetCode:   sellingCode,
			SellingAssetIssuer: sellingIssuer,
		},
		OfferID: int64(manageSellOffer.OfferId),
	}, nil
}
