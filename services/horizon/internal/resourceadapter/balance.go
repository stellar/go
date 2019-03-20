package resourceadapter

import (
	"github.com/stellar/go/amount"
	. "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/assets"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/xdr"
)

func PopulateBalance(dest *Balance, row core.Trustline) (err error) {
	dest.Type, err = assets.String(row.Assettype)
	if err != nil {
		return
	}

	dest.Balance = amount.String(row.Balance)
	dest.BuyingLiabilities = amount.String(row.BuyingLiabilities)
	dest.SellingLiabilities = amount.String(row.SellingLiabilities)
	dest.Limit = amount.String(row.Tlimit)
	dest.Issuer = row.Issuer
	dest.Code = row.Assetcode
	dest.LastModifiedLedger = row.LastModified
	isAuthorized := row.IsAuthorized()
	dest.IsAuthorized = &isAuthorized
	return
}

func PopulateNativeBalance(dest *Balance, stroops, buyingLiabilities, sellingLiabilities xdr.Int64) (err error) {
	dest.Type, err = assets.String(xdr.AssetTypeAssetTypeNative)
	if err != nil {
		return
	}

	dest.Balance = amount.String(stroops)
	dest.BuyingLiabilities = amount.String(buyingLiabilities)
	dest.SellingLiabilities = amount.String(sellingLiabilities)
	dest.LastModifiedLedger = 0
	dest.Limit = ""
	dest.Issuer = ""
	dest.Code = ""
	dest.IsAuthorized = nil
	return
}
