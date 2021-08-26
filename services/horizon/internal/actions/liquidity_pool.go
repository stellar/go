package actions

import (
	"net/http"

	protocol "github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
)

// GetLiquidityPoolByIDHandler is the action handler for all end-points returning a liquidity pool.
type GetLiquidityPoolByIDHandler struct{}

// LiquidityPoolQuery query struct for liquidity_pools/id endpoint
type LiquidityPoolQuery struct {
	ID string `schema:"id" valid:"sha256"`
}

// GetResource returns an claimable balance page.
func (handler GetLiquidityPoolByIDHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()
	qp := LiquidityPoolQuery{}
	err := getParams(&qp, r)
	if err != nil {
		return nil, err
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}
	cb, err := historyQ.FindLiquidityPoolByID(ctx, qp.ID)
	if err != nil {
		return nil, err
	}
	ledger := &history.Ledger{}
	err = historyQ.LedgerBySequence(ctx, ledger, int32(cb.LastModifiedLedger))
	if historyQ.NoRows(err) {
		ledger = nil
	} else if err != nil {
		return nil, errors.Wrap(err, "LedgerBySequence error")
	}

	var resource protocol.LiquidityPool
	err = resourceadapter.PopulateLiquidityPool(ctx, &resource, cb, ledger)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
