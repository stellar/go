package actions

import (
	"context"
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
)

// AccountOffersQuery query struct for offers end-point
type OfferByIDQuery struct {
	OfferID uint64 `schema:"offer_id" valid:"-"`
}

// GetOfferByID is the action handler for the /offers/{id} endpoint
type GetOfferByID struct{}

// GetResource returns an offer by id.
func (handler GetOfferByID) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()

	qp := OfferByIDQuery{}
	if err := getParams(&qp, r); err != nil {
		return nil, err
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	record, err := historyQ.GetOfferByID(int64(qp.OfferID))
	if err != nil {
		return nil, err
	}

	ledger := &history.Ledger{}
	err = historyQ.LedgerBySequence(
		ledger,
		int32(record.LastModifiedLedger),
	)
	if historyQ.NoRows(err) {
		ledger = nil
	} else if err != nil {
		return nil, err
	}

	var offerResponse horizon.Offer
	resourceadapter.PopulateOffer(ctx, &offerResponse, record, ledger)
	return offerResponse, nil
}

// OffersQuery query struct for offers end-point
type OffersQuery struct {
	SellingBuyingAssetQueryParams `valid:"-"`
	Seller                        string `schema:"seller" valid:"accountID,optional"`
}

// URITemplate returns a rfc6570 URI template the query struct
func (q OffersQuery) URITemplate() string {
	// building this manually since we don't want to include all the params in SellingBuyingAssetQueryParams
	return "/offers{?selling,buying,seller,cursor,limit,order}"
}

// Validate runs custom validations.
func (q OffersQuery) Validate() error {
	return q.SellingBuyingAssetQueryParams.Validate()
}

// GetOffersHandler is the action handler for the /offers endpoint
type GetOffersHandler struct {
}

// GetResourcePage returns a page of offers.
func (handler GetOffersHandler) GetResourcePage(
	w HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	ctx := r.Context()
	qp := OffersQuery{}
	err := getParams(&qp, r)
	if err != nil {
		return nil, err
	}

	pq, err := GetPageQuery(r)
	if err != nil {
		return nil, err
	}

	selling, err := qp.Selling()
	if err != nil {
		return nil, err
	}
	buying, err := qp.Buying()
	if err != nil {
		return nil, err
	}

	query := history.OffersQuery{
		PageQuery: pq,
		SellerID:  qp.Seller,
		Selling:   selling,
		Buying:    buying,
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	offers, err := getOffersPage(ctx, historyQ, query)
	if err != nil {
		return nil, err
	}

	return offers, nil
}

// AccountOffersQuery query struct for offers end-point
type AccountOffersQuery struct {
	AccountID string `schema:"account_id" valid:"accountID,required"`
}

// GetAccountOffersHandler is the action handler for the
// `/accounts/{account_id}/offers` endpoint when using experimental ingestion.
type GetAccountOffersHandler struct {
}

func (handler GetAccountOffersHandler) parseOffersQuery(r *http.Request) (history.OffersQuery, error) {
	pq, err := GetPageQuery(r)
	if err != nil {
		return history.OffersQuery{}, err
	}

	qp := AccountOffersQuery{}
	if err = getParams(&qp, r); err != nil {
		return history.OffersQuery{}, err
	}

	query := history.OffersQuery{
		PageQuery: pq,
		SellerID:  qp.AccountID,
	}

	return query, nil
}

// GetResourcePage returns a page of offers for a given account.
func (handler GetAccountOffersHandler) GetResourcePage(
	w HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	ctx := r.Context()
	query, err := handler.parseOffersQuery(r)
	if err != nil {
		return nil, err
	}

	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	offers, err := getOffersPage(ctx, historyQ, query)
	if err != nil {
		return nil, err
	}

	return offers, nil
}

func getOffersPage(ctx context.Context, historyQ *history.Q, query history.OffersQuery) ([]hal.Pageable, error) {
	records, err := historyQ.GetOffers(query)
	if err != nil {
		return nil, err
	}

	ledgerCache := history.LedgerCache{}
	for _, record := range records {
		ledgerCache.Queue(int32(record.LastModifiedLedger))
	}

	if err := ledgerCache.Load(historyQ); err != nil {
		return nil, errors.Wrap(err, "failed to load ledger batch")
	}

	var offers []hal.Pageable
	for _, record := range records {
		var offerResponse horizon.Offer

		var ledger *history.Ledger
		if l, ok := ledgerCache.Records[int32(record.LastModifiedLedger)]; ok {
			ledger = &l
		}

		resourceadapter.PopulateOffer(ctx, &offerResponse, record, ledger)
		offers = append(offers, offerResponse)
	}

	return offers, nil
}
