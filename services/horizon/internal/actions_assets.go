package horizon

import (
	"math"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/render/hal"
	"github.com/stellar/go/services/horizon/internal/resource"
)

// This file contains the actions:
//
// AssetsAction: pages of assets

// AssetsAction renders a page of Assets
type AssetsAction struct {
	Action
	AssetCode    string
	AssetIssuer  string
	PagingParams db2.PageQuery
	Records      []resource.JoinedAssetStat
	Page         hal.Page
}

// JSON is a method for actions.JSON
func (action *AssetsAction) JSON() {
	action.Do(
		action.loadParams,
		action.loadRecord,
		action.loadPage,
		func() {
			hal.Render(action.W, action.Page)
		},
	)
}

func (action *AssetsAction) loadParams() {
	action.AssetCode = action.GetString("asset_code")
	action.AssetIssuer = action.GetString("asset_issuer")
	action.PagingParams = action.GetPageQuery()
}

// history_assets (id) is 32-bits but paging expects to query on an int64 so defaulting for now
func defaultDescendingCursor(action *AssetsAction) {
	if action.PagingParams.Cursor == "" && action.PagingParams.Order == db2.OrderDescending {
		action.PagingParams.Cursor = strconv.FormatInt(math.MaxInt32, 10)
	}
}

func (action *AssetsAction) loadRecord() {
	sql := sq.Select(
		"hist.id",
		"hist.asset_type",
		"hist.asset_code",
		"hist.asset_issuer",
		"stats.amount",
		"stats.num_accounts",
		"stats.flags",
		"stats.toml",
	).From("history_assets hist").Join("asset_stats stats ON hist.id = stats.id")

	if action.AssetCode != "" {
		sql = sql.Where("hist.asset_code = ?", action.AssetCode)
	}
	if action.AssetIssuer != "" {
		sql = sql.Where("hist.asset_issuer = ?", action.AssetIssuer)
	}

	defaultDescendingCursor(action)
	sql, action.Err = action.PagingParams.ApplyTo(sql, "hist.id")
	if action.Err != nil {
		return
	}

	action.Err = action.HistoryQ().Select(&action.Records, sql)
}

func (action *AssetsAction) loadPage() {
	for _, record := range action.Records {
		var res resource.AssetStat
		res.Populate(action.Ctx, record)
		action.Page.Add(res)
	}

	action.Page.BaseURL = action.BaseURL()
	action.Page.BasePath = action.Path()
	action.Page.Limit = action.PagingParams.Limit
	action.Page.Cursor = action.PagingParams.Cursor
	action.Page.Order = action.PagingParams.Order

	var linkParams []*hal.LinkParam
	if action.AssetCode != "" {
		linkParams = append(linkParams, &hal.LinkParam{
			Key:   "asset_code",
			Value: action.AssetCode,
		})
	}
	if action.AssetIssuer != "" {
		linkParams = append(linkParams, &hal.LinkParam{
			Key:   "asset_issuer",
			Value: action.AssetIssuer,
		})
	}
	action.Page.PopulateLinks(linkParams...)
}
