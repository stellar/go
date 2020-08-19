package actions

import (
	"fmt"
	"net/http"

	protocol "github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// GetClaimableBalanceByIDHandler is the action handler for all end-points returning a claimable balance.
type GetClaimableBalanceByIDHandler struct{}

// ClaimableBalanceQuery query struct for claimables_balances/id end-point
type ClaimableBalanceQuery struct {
	ID string `schema:"id" valid:"-"`
}

// Validate validates the balance id
func (q ClaimableBalanceQuery) Validate() error {
	if _, err := q.BalanceID(); err != nil {
		return err
	}
	return nil
}

// BalanceID returns the xdr.ClaimableBalanceId from the request query
func (q ClaimableBalanceQuery) BalanceID() (xdr.ClaimableBalanceId, error) {
	var balanceID xdr.ClaimableBalanceId
	err := xdr.SafeUnmarshalHex(q.ID, &balanceID)
	if err != nil {
		return balanceID, problem.MakeInvalidFieldProblem(
			"id",
			fmt.Errorf("Invalid claimable balance ID"),
		)
	}
	return balanceID, nil
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
	balanceID, err := qp.BalanceID()
	if err != nil {
		return nil, err
	}
	cb, err := historyQ.FindClaimableBalanceByID(balanceID)
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
