package operation

import (
	"fmt"
	"strconv"
)

type ManageBuyOffer struct {
	OfferID            int64   `json:"offer_id,string"`
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

func (o *LedgerOperation) ManageBuyOfferDetails() (ManageBuyOffer, error) {
	op, ok := o.Operation.Body.GetManageBuyOfferOp()
	if !ok {
		return ManageBuyOffer{}, fmt.Errorf("could not access ManageBuyOffer info for this operation (index %d)", o.OperationIndex)
	}

	manageBuyOffer := ManageBuyOffer{
		OfferID: int64(op.OfferId),
		Amount:  int64(op.BuyAmount),
		PriceN:  int32(op.Price.N),
		PriceD:  int32(op.Price.D),
	}

	var err error
	manageBuyOffer.Price, err = strconv.ParseFloat(op.Price.String(), 64)
	if err != nil {
		return ManageBuyOffer{}, err
	}

	var buyingAssetCode, buyingAssetIssuer, buyingAssetType string
	err = op.Buying.Extract(&buyingAssetType, &buyingAssetCode, &buyingAssetIssuer)
	if err != nil {
		return ManageBuyOffer{}, err
	}

	manageBuyOffer.BuyingAssetCode = buyingAssetCode
	manageBuyOffer.BuyingAssetIssuer = buyingAssetIssuer
	manageBuyOffer.BuyingAssetType = buyingAssetType

	var sellingAssetCode, sellingAssetIssuer, sellingAssetType string
	err = op.Selling.Extract(&sellingAssetType, &sellingAssetCode, &sellingAssetIssuer)
	if err != nil {
		return ManageBuyOffer{}, err
	}

	manageBuyOffer.SellingAssetCode = sellingAssetCode
	manageBuyOffer.SellingAssetIssuer = sellingAssetIssuer
	manageBuyOffer.SellingAssetType = sellingAssetType

	return manageBuyOffer, nil
}
