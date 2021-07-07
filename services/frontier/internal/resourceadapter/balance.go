package resourceadapter

import (
	"github.com/xdbfoundation/go/amount"
	protocol "github.com/xdbfoundation/go/protocols/frontier"
	"github.com/xdbfoundation/go/services/frontier/internal/assets"
	"github.com/xdbfoundation/go/services/frontier/internal/db2/history"
	"github.com/xdbfoundation/go/support/errors"
	"github.com/xdbfoundation/go/xdr"
)

func PopulateBalance(dest *protocol.Balance, row history.TrustLine) (err error) {
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
	if row.Sponsor.Valid {
		dest.Sponsor = row.Sponsor.String
	}
	return
}

func PopulateNativeBalance(dest *protocol.Balance, nibbs, buyingLiabilities, sellingLiabilities xdr.Int64) (err error) {
	dest.Type, err = assets.String(xdr.AssetTypeAssetTypeNative)
	if err != nil {
		return errors.Wrap(err, "getting the string representation from the provided xdr asset type")
	}

	dest.Balance = amount.String(nibbs)
	dest.BuyingLiabilities = amount.String(buyingLiabilities)
	dest.SellingLiabilities = amount.String(sellingLiabilities)
	dest.LastModifiedLedger = 0
	dest.Limit = ""
	dest.Issuer = ""
	dest.Code = ""
	dest.IsAuthorized = nil
	dest.IsAuthorizedToMaintainLiabilities = nil
	return
}
