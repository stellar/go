package resourceadapter

import (
	"context"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/services/horizon/internal/db2/assets"
	"github.com/stellar/go/xdr"
	. "github.com/stellar/go/protocols/resource"
	"github.com/stellar/go/support/render/hal"
)

// PopulateAssetStat fills out the details
func PopulateAssetStat(
	ctx context.Context,
	dest *AssetStat,
	row assets.AssetStatsR,
) (err error) {

	dest.Asset.Type = row.Type
	dest.Asset.Code = row.Code
	dest.Asset.Issuer = row.Issuer
	dest.Amount = amount.StringFromInt64(row.Amount)
	dest.NumAccounts = row.NumAccounts
	dest.Flags = AccountFlags{
		(row.Flags & int8(xdr.AccountFlagsAuthRequiredFlag)) != 0,
		(row.Flags & int8(xdr.AccountFlagsAuthRevocableFlag)) != 0,
	}
	dest.PT = row.SortKey

	dest.Links.Toml = hal.NewLink(row.Toml)
	return
}

