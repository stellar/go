package horizon

import (
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/support/render/hal"
)

// Interface verification
var _ actions.JSONer = (*PathIndexAction)(nil)

// PathIndexAction provides path finding
type PathIndexAction struct {
	Action
	Query   paths.Query
	Records []paths.Path
	Page    hal.BasePage
}

// JSON implements actions.JSON
func (action *PathIndexAction) JSON() error {
	action.Do(
		action.loadQuery,
		action.loadSourceAssets,
		action.loadRecords,
		action.loadPage,
		func() { hal.Render(action.W, action.Page) },
	)
	return action.Err
}

func (action *PathIndexAction) loadQuery() {
	action.Query.DestinationAmount = action.GetPositiveAmount("destination_amount")
	action.Query.DestinationAsset = action.GetAsset("destination_")
	action.Query.SourceAccount = action.Base.GetAccountID("source_account")
}

func (action *PathIndexAction) loadSourceAssets() {
	action.Query.SourceAssets, action.Query.SourceAssetBalances, action.Err = action.CoreQ().AssetsForAddress(
		action.Query.SourceAccount.Address(),
	)
}

func (action *PathIndexAction) loadRecords() {
	action.Records, action.Err = action.App.paths.Find(action.Query, action.App.config.MaxPathLength)
	if action.Err == simplepath.ErrEmptyInMemoryOrderBook {
		action.Err = problem.StillIngesting
	}
}

func (action *PathIndexAction) loadPage() {
	action.Page.Init()
	for _, p := range action.Records {
		var res horizon.Path
		action.Err = resourceadapter.PopulatePath(action.R.Context(), &res, p)

		if action.Err != nil {
			return
		}
		action.Page.Add(res)
	}
}
