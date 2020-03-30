package resourceadapter

import (
	"github.com/stellar/go/amount"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/assets"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func PopulateBalance(dest *protocol.Balance, row core.Trustline) (err error) {
	dest.Type, err = assets.String(row.Assettype)
	if err != nil {
		return errors.Wrap(err, "getting the string representation from the provided xdr asset type")
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
	dest.IsAuthorizedToMaintainLiabilities = &isAuthorized
	isAuthorizedToMaintainLiabilities := row.IsAuthorizedToMaintainLiabilities()
	if isAuthorizedToMaintainLiabilities {
		dest.IsAuthorizedToMaintainLiabilities = &isAuthorizedToMaintainLiabilities
	}
	return
}

func PopulateHistoryBalance(dest *protocol.Balance, row history.TrustLine) (err error) {
	dest.Type, err = assets.String(row.AssetType)
	if err != nil {
		return errors.Wrap(err, "getting the string representation from the provided xdr asset type")
	}

	dest.Balance = amount.StringFromInt64(row.Balance)
	dest.BuyingLiabilities = amount.StringFromInt64(row.BuyingLiabilities)
	dest.SellingLiabilities = amount.StringFromInt64(row.SellingLiabilities)
	dest.Limit = amount.StringFromInt64(row.Limit)
	dest.Issuer = row.AssetIssuer
	dest.Code = row.AssetCode
	dest.LastModifiedLedger = row.LastModifiedLedger
	isAuthorized := row.IsAuthorized()
	dest.IsAuthorized = &isAuthorized
	dest.IsAuthorizedToMaintainLiabilities = &isAuthorized
	isAuthorizedToMaintainLiabilities := row.IsAuthorizedToMaintainLiabilities()
	if isAuthorizedToMaintainLiabilities {
		dest.IsAuthorizedToMaintainLiabilities = &isAuthorizedToMaintainLiabilities
	}
	return
}

func PopulateNativeBalance(dest *protocol.Balance, stroops, buyingLiabilities, sellingLiabilities xdr.Int64) (err error) {
	dest.Type, err = assets.String(xdr.AssetTypeAssetTypeNative)
	if err != nil {
		return errors.Wrap(err, "getting the string representation from the provided xdr asset type")
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
