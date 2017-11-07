package resource

import (
	"strconv"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/render/hal"
	"github.com/stellar/go/xdr"
	"golang.org/x/net/context"
)

// JoinedAssetStat is the db representation of a join result from Asset and AssetStat
type JoinedAssetStat struct {
	ingest.AssetStat
	history.Asset
}

// Populate fills out the details
func (res *AssetStat) Populate(
	ctx context.Context,
	row JoinedAssetStat,
) (err error) {

	res.Asset.Type = row.Type
	res.Asset.Code = row.Code
	res.Asset.Issuer = row.Issuer
	res.Amount = row.Amount
	res.NumAccounts = row.NumAccounts
	res.Flags = AccountFlags{
		(row.Flags & int8(xdr.AccountFlagsAuthRequiredFlag)) != 0,
		(row.Flags & int8(xdr.AccountFlagsAuthRevocableFlag)) != 0,
	}
	res.PT = strconv.FormatInt(row.Asset.ID, 10)

	res.Links.Toml = hal.NewLink(row.Toml)
	return
}

// PagingToken implementation for hal.Pageable
func (res AssetStat) PagingToken() string {
	return res.PT
}
