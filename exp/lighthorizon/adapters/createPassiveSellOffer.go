package adapters

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
)

func populateCreatePassiveSellOfferOperation(op *common.Operation, baseOp operations.Base) (operations.CreatePassiveSellOffer, error) {
	createPassiveSellOffer := op.Get().Body.MustCreatePassiveSellOfferOp()
	baseOp.Type = "create_passive_sell_offer"

	var (
		buyingAssetType string
		buyingCode      string
		buyingIssuer    string
	)
	err := createPassiveSellOffer.Buying.Extract(&buyingAssetType, &buyingCode, &buyingIssuer)
	if err != nil {
		return operations.CreatePassiveSellOffer{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	var (
		sellingAssetType string
		sellingCode      string
		sellingIssuer    string
	)
	err = createPassiveSellOffer.Selling.Extract(&sellingAssetType, &sellingCode, &sellingIssuer)
	if err != nil {
		return operations.CreatePassiveSellOffer{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	return operations.CreatePassiveSellOffer{
		Offer: operations.Offer{
			Base:   baseOp,
			Amount: amount.String(createPassiveSellOffer.Amount),
			Price:  createPassiveSellOffer.Price.String(),
			PriceR: base.Price{
				N: int32(createPassiveSellOffer.Price.N),
				D: int32(createPassiveSellOffer.Price.D),
			},
			BuyingAssetType:    buyingAssetType,
			BuyingAssetCode:    buyingCode,
			BuyingAssetIssuer:  buyingIssuer,
			SellingAssetType:   sellingAssetType,
			SellingAssetCode:   sellingCode,
			SellingAssetIssuer: sellingIssuer,
		},
	}, nil
}
