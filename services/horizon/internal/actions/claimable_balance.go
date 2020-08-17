package actions

import (
	"net/http"

	protocol "github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
)

// GetClaimableBalanceByIDHandler is the action handler for all end-points returning a claimable balance.
type GetClaimableBalanceByIDHandler struct{}

// ClaimableBalanceQuery query struct for claimables_balances/id end-point
type ClaimableBalanceQuery struct {
	// TODO add validation - let's postpone it until the final representation of ID is defined.
	ID string `schema:"id" valid:"-"`
}

// GetResource returns an claimable balance page.
func (handler GetClaimableBalanceByIDHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()
	qp := ClaimableBalanceQuery{}
	err := getParams(&qp, r)
	if err != nil {
		return nil, err
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}
	cb, err := historyQ.FindClaimableBalanceByID(qp.ID)
	if err != nil {
		return nil, err
	}

	var resource protocol.ClaimableBalance
	err = resourceadapter.PopulateClaimableBalance(ctx, &resource, cb)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
