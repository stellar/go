package operation

import (
	"fmt"
	"strconv"
)

type CreatePassiveSellOfferDetail struct {
	Amount             int64   `json:"amount,string"`
	PriceN             int32   `json:"price_n"`
	PriceD             int32   `json:"price_d"`
	Price              float64 `json:"price"`
	BuyingAssetCode    string  `json:"buying_asset_code"`
	BuyingAssetIssuer  string  `json:"buying_asset_issuer"`
	BuyingAssetType    string  `json:"buying_asset_type"`
	SellingAssetCode   string  `json:"selling_asset_code"`
	SellingAssetIssuer string  `json:"selling_asset_issuer"`
	SellingAssetType   string  `json:"selling_asset_type"`
}

func (o *LedgerOperation) CreatePassiveSellOfferDetails() (CreatePassiveSellOfferDetail, error) {
	op, ok := o.Operation.Body.GetCreatePassiveSellOfferOp()
	if !ok {
		return CreatePassiveSellOfferDetail{}, fmt.Errorf("could not access CreatePassiveSellOffer info for this operation (index %d)", o.OperationIndex)
	}

	createPassiveSellOffer := CreatePassiveSellOfferDetail{
		Amount: int64(op.Amount),
		PriceN: int32(op.Price.N),
		PriceD: int32(op.Price.D),
	}

	var err error
	createPassiveSellOffer.Price, err = strconv.ParseFloat(op.Price.String(), 64)
	if err != nil {
		return CreatePassiveSellOfferDetail{}, err
	}

	var buyingAssetCode, buyingAssetIssuer, buyingAssetType string
	err = op.Buying.Extract(&buyingAssetType, &buyingAssetCode, &buyingAssetIssuer)
	if err != nil {
		return CreatePassiveSellOfferDetail{}, err
	}

	createPassiveSellOffer.BuyingAssetCode = buyingAssetCode
	createPassiveSellOffer.BuyingAssetIssuer = buyingAssetIssuer
	createPassiveSellOffer.BuyingAssetType = buyingAssetType

	var sellingAssetCode, sellingAssetIssuer, sellingAssetType string
	err = op.Selling.Extract(&sellingAssetType, &sellingAssetCode, &sellingAssetIssuer)
	if err != nil {
		return CreatePassiveSellOfferDetail{}, err
	}

	createPassiveSellOffer.SellingAssetCode = sellingAssetCode
	createPassiveSellOffer.SellingAssetIssuer = sellingAssetIssuer
	createPassiveSellOffer.SellingAssetType = sellingAssetType

	return createPassiveSellOffer, nil
}
