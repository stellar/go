package horizon

import (
	"context"
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/paths"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	horizonProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// FindPathsHandler is the http handler for the find payment paths endpoint
type FindPathsHandler struct {
	staleThreshold      uint
	maxPathLength       uint
	checkHistoryIsStale bool
	pathFinder          paths.Finder
	coreQ               *core.Q
}

// ServeHTTP implements the http.Handler interface
func (handler FindPathsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if handler.checkHistoryIsStale && isHistoryStale(handler.staleThreshold) {
		ls := ledger.CurrentState()
		err := hProblem.StaleHistory
		err.Extras = map[string]interface{}{
			"history_latest_ledger": ls.HistoryLatest,
			"core_latest_ledger":    ls.CoreLatest,
		}
		problem.Render(ctx, w, err)
		return
	}

	query := paths.Query{}
	var err error
	query.DestinationAmount, err = actions.GetPositiveAmount(r, "destination_amount")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	var sourceAccount string
	if sourceAccount, err = getAccountID(r, "source_account", true); err != nil {
		problem.Render(ctx, w, err)
		return
	}
	query.SourceAccount = xdr.MustAddress(sourceAccount)

	if query.DestinationAsset, err = actions.GetAsset(r, "destination_"); err != nil {
		problem.Render(ctx, w, err)
		return
	}

	query.SourceAssets, query.SourceAssetBalances, err = handler.coreQ.AssetsForAddress(
		query.SourceAccount.Address(),
	)
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	records := []paths.Path{}
	if len(query.SourceAssets) > 0 {
		records, err = handler.pathFinder.Find(query, handler.maxPathLength)
		if err == simplepath.ErrEmptyInMemoryOrderBook {
			err = horizonProblem.StillIngesting
		}
	}
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	renderPaths(ctx, records, w)
}

func renderPaths(ctx context.Context, records []paths.Path, w http.ResponseWriter) {
	var page hal.BasePage
	page.Init()
	for _, p := range records {
		var res horizon.Path
		err := resourceadapter.PopulatePath(ctx, &res, p)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}
		page.Add(res)
	}
	httpjson.Render(w, page, httpjson.HALJSON)
}

// FindFixedPathsHandler is the http handler for the find fixed payment paths endpoint
// Fixed payment paths are payment paths where both the source and destination asset are fixed
type FindFixedPathsHandler struct {
	maxPathLength uint
	pathFinder    paths.Finder
}

// ServeHTTP implements the http.Handler interface
func (handler FindFixedPathsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sourceAccount, err := actions.GetString(r, "source_account")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	var sourceAccountID *xdr.AccountId
	if sourceAccount != "" {
		var accountID xdr.AccountId
		if accountID, err = actions.GetAccountID(r, "source_account"); err != nil {
			problem.Render(ctx, w, err)
			return
		} else {
			sourceAccountID = &accountID
		}
	}

	destinationAsset, err := actions.GetAsset(r, "destination_")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	sourceAsset, err := actions.GetAsset(r, "source_")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	amountToSpend, err := actions.GetPositiveAmount(r, "source_amount")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	records, err := handler.pathFinder.FindFixedPaths(
		sourceAccountID,
		sourceAsset,
		amountToSpend,
		destinationAsset,
		handler.maxPathLength,
	)
	if err == simplepath.ErrEmptyInMemoryOrderBook {
		err = horizonProblem.StillIngesting
	}
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	renderPaths(ctx, records, w)
}
