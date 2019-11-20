package actions

import (
	"github.com/stellar/go/xdr"
)

// SellingBuyingAssetQueryParams query struct for end-points requiring a selling
// and buying asset
type SellingBuyingAssetQueryParams struct {
	SellingAssetType   string `schema:"selling_asset_type" valid:"assetType,optional"`
	SellingAssetIssuer string `schema:"selling_asset_issuer" valid:"accountID,optional"`
	SellingAssetCode   string `schema:"selling_asset_code" valid:"-"`
	BuyingAssetType    string `schema:"buying_asset_type" valid:"assetType,optional"`
	BuyingAssetIssuer  string `schema:"buying_asset_issuer" valid:"accountID,optional"`
	BuyingAssetCode    string `schema:"buying_asset_code" valid:"-"`
}

// Validate runs custom validations buying and selling
func (q SellingBuyingAssetQueryParams) Validate() error {
	err := ValidateAssetParams(q.SellingAssetType, q.SellingAssetCode, q.SellingAssetIssuer, "selling_")
	if err != nil {
		return err
	}
	err = ValidateAssetParams(q.BuyingAssetType, q.BuyingAssetCode, q.BuyingAssetIssuer, "buying_")
	if err != nil {
		return err
	}
	return nil
}

// Selling returns an xdr.Asset representing the selling side of the offer.
func (q SellingBuyingAssetQueryParams) Selling() *xdr.Asset {
	if len(q.SellingAssetType) == 0 {
		return nil
	}

	selling, err := xdr.BuildAsset(
		q.SellingAssetType,
		q.SellingAssetIssuer,
		q.SellingAssetCode,
	)

	if err != nil {
		panic(err)
	}

	return &selling
}

// Buying returns an *xdr.Asset representing the buying side of the offer.
func (q SellingBuyingAssetQueryParams) Buying() *xdr.Asset {
	if len(q.BuyingAssetType) == 0 {
		return nil
	}

	buying, err := xdr.BuildAsset(
		q.BuyingAssetType,
		q.BuyingAssetIssuer,
		q.BuyingAssetCode,
	)

	if err != nil {
		panic(err)
	}

	return &buying
}
