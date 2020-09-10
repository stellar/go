package actions

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stellar/go/protocols/horizon"
	protocol "github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/render/hal"
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

// ClaimableBalancesQuery query struct for claimable_balances end-point
type ClaimableBalancesQuery struct {
	AssetFilter    string `schema:"asset" valid:"asset,optional"`
	SponsorFilter  string `schema:"sponsor" valid:"accountID,optional"`
	ClaimantFilter string `schema:"claimant" valid:"accountID,optional"`
}

func (q ClaimableBalancesQuery) asset() *xdr.Asset {
	if len(q.AssetFilter) > 0 {
		switch q.AssetFilter {
		case "native":
			asset := xdr.MustNewNativeAsset()
			return &asset
		default:
			parts := strings.Split(q.AssetFilter, ":")
			asset := xdr.MustNewCreditAsset(parts[0], parts[1])
			return &asset
		}
	}
	return nil
}

func (q ClaimableBalancesQuery) sponsor() *xdr.AccountId {
	if q.SponsorFilter != "" {
		return xdr.MustAddressPtr(q.SponsorFilter)
	}
	return nil
}

func (q ClaimableBalancesQuery) claimant() *xdr.AccountId {
	if q.ClaimantFilter != "" {
		return xdr.MustAddressPtr(q.ClaimantFilter)
	}
	return nil
}

// URITemplate returns a rfc6570 URI template the query struct
func (q ClaimableBalancesQuery) URITemplate() string {
	return "/claimable_balances?{asset,claimant,sponsor}"
}

type GetClaimableBalancesHandler struct {
}

// GetResourcePage returns a page of claimable balances.
func (handler GetClaimableBalancesHandler) GetResourcePage(
	w HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	ctx := r.Context()
	qp := ClaimableBalancesQuery{}
	err := getParams(&qp, r)
	if err != nil {
		return nil, err
	}

	pq, err := GetPageQuery(r, DisableCursorValidation)
	if err != nil {
		return nil, err
	}

	query := history.ClaimableBalancesQuery{
		PageQuery: pq,
		Asset:     qp.asset(),
		Sponsor:   qp.sponsor(),
		Claimant:  qp.claimant(),
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	claimableBalances, err := getClaimableBalancesPage(ctx, historyQ, query)
	if err != nil {
		return nil, err
	}

	return claimableBalances, nil
}

func getClaimableBalancesPage(ctx context.Context, historyQ *history.Q, query history.ClaimableBalancesQuery) ([]hal.Pageable, error) {
	records, err := historyQ.GetClaimableBalances(query)
	if err != nil {
		return nil, err
	}

	var claimableBalances []hal.Pageable
	for _, record := range records {
		var response horizon.ClaimableBalance

		resourceadapter.PopulateClaimableBalance(ctx, &response, record)
		claimableBalances = append(claimableBalances, response)
	}

	return claimableBalances, nil
}
