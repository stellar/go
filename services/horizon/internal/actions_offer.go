package horizon

import (
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// This file contains the actions:

// Interface verifications
var _ actions.JSONer = (*OffersByAccountAction)(nil)
var _ actions.EventStreamer = (*OffersByAccountAction)(nil)

// OffersByAccountAction renders a page of offer resources, for a given
// account.  These offers are present in the ledger as of the latest validated
// ledger.
type OffersByAccountAction struct {
	Action
	Address   string
	PageQuery db2.PageQuery
	Records   []core.Offer
	Ledgers   *history.LedgerCache
	Page      hal.Page
}

// JSON is a method for actions.JSON
func (action *OffersByAccountAction) JSON() error {
	action.Do(
		action.loadParams,
		action.loadRecords,
		action.loadLedgers,
		action.loadPage,
		func() { hal.Render(action.W, action.Page) },
	)
	return action.Err
}

// SSE is a method for actions.SSE
func (action *OffersByAccountAction) SSE(stream *sse.Stream) error {
	// Load the page query params the first time SSE() is called. We update
	// the pagination cursor below before sending each event to the stream.
	if action.PageQuery.Cursor == "" {
		action.loadParams()
		if action.Err != nil {
			return action.Err
		}
	}

	action.Do(
		action.loadRecords,
		action.loadLedgers,
		func() {
			stream.SetLimit(int(action.PageQuery.Limit))
			for _, record := range action.Records {
				ledger, found := action.Ledgers.Records[record.Lastmodified]
				ledgerPtr := &ledger
				if !found {
					ledgerPtr = nil
				}
				var res horizon.Offer
				resourceadapter.PopulateOffer(action.R.Context(), &res, record, ledgerPtr)
				action.PageQuery.Cursor = res.PagingToken()
				stream.Send(sse.Event{ID: res.PagingToken(), Data: res})
			}
		},
	)

	return action.Err
}

func (action *OffersByAccountAction) loadParams() {
	action.PageQuery = action.GetPageQuery()
	action.Address = action.GetAddress("account_id")
}

// loadLedgers populates the ledger cache for this action
func (action *OffersByAccountAction) loadLedgers() {
	action.Ledgers = &history.LedgerCache{}

	for _, offer := range action.Records {
		action.Ledgers.Queue(offer.Lastmodified)
	}
	action.Err = action.Ledgers.Load(action.HistoryQ())
}

func (action *OffersByAccountAction) loadRecords() {
	action.Err = action.CoreQ().OffersByAddress(
		&action.Records,
		action.Address,
		action.PageQuery,
	)
}

func (action *OffersByAccountAction) loadPage() {
	for _, record := range action.Records {
		ledger, found := action.Ledgers.Records[record.Lastmodified]
		ledgerPtr := &ledger
		if !found {
			ledgerPtr = nil
		}

		var res horizon.Offer
		resourceadapter.PopulateOffer(action.R.Context(), &res, record, ledgerPtr)
		action.Page.Add(res)
	}

	action.Page.FullURL = action.FullURL()
	action.Page.Limit = action.PageQuery.Limit
	action.Page.Cursor = action.PageQuery.Cursor
	action.Page.Order = action.PageQuery.Order
	action.Page.PopulateLinks()
}

// GetOffersHandle is the http handler for the /offers endpoint
type GetOffersHandle struct {
	historyQ *history.Q
}

// ServeHTTP implements the http.Handler interface
func (handler GetOffersHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pq, err := actions.GetPageQuery(r)

	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	seller, err := actions.GetString(r, "seller")

	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	var selling *xdr.Asset
	sellingAsset, found := actions.MaybeGetAsset(r, "selling_")

	if found {
		selling = &sellingAsset
	}

	var buying *xdr.Asset
	buyingAsset, found := actions.MaybeGetAsset(r, "buying_")

	if found {
		buying = &buyingAsset
	}

	query := history.OffersQuery{
		PageQuery: pq,
		SellerID:  seller,
		Selling:   selling,
		Buying:    buying,
	}

	records, err := handler.historyQ.GetOffers(query)

	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	page := hal.Page{
		Cursor: pq.Cursor,
		Order:  pq.Order,
		Limit:  pq.Limit,
	}

	ledgerCache := history.LedgerCache{}
	for _, record := range records {
		ledgerCache.Queue(int32(record.LastModifiedLedger))
	}

	err = ledgerCache.Load(handler.historyQ)

	if err != nil {
		problem.Render(ctx, w, errors.Wrap(err, "failed to load ledger batch"))
		return
	}

	for _, record := range records {
		var offerResponse horizon.Offer

		ledger, found := ledgerCache.Records[int32(record.LastModifiedLedger)]
		ledgerPtr := &ledger
		if !found {
			ledgerPtr = nil
		}

		resourceadapter.PopulateHistoryOffer(ctx, &offerResponse, record, ledgerPtr)
		page.Add(offerResponse)
	}

	page.FullURL = actions.FullURL(ctx)
	page.PopulateLinks()
	httpjson.Render(w, page, httpjson.HALJSON)
}

// GetAccountOffersHandle is the http handler for the /accounts/{account_id}/offers endpoint when using experimental ingestion.
type GetAccountOffersHandle struct {
	historyQ *history.Q
}

// ServeHTTP implements the http.Handler interface
func (handler GetAccountOffersHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pq, err := actions.GetPageQuery(r)

	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	seller, err := actions.GetString(r, "account_id")

	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	query := history.OffersQuery{
		PageQuery: pq,
		SellerID:  seller,
		Selling:   nil,
		Buying:    nil,
	}

	records, err := handler.historyQ.GetOffers(query)

	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	page := hal.Page{
		Cursor: pq.Cursor,
		Order:  pq.Order,
		Limit:  pq.Limit,
	}

	ledgerCache := history.LedgerCache{}
	for _, record := range records {
		ledgerCache.Queue(int32(record.LastModifiedLedger))
	}

	err = ledgerCache.Load(handler.historyQ)

	if err != nil {
		problem.Render(ctx, w, errors.Wrap(err, "failed to load ledger batch"))
		return
	}

	for _, record := range records {
		var offerResponse horizon.Offer

		ledger, found := ledgerCache.Records[int32(record.LastModifiedLedger)]
		ledgerPtr := &ledger
		if !found {
			ledgerPtr = nil
		}

		resourceadapter.PopulateHistoryOffer(ctx, &offerResponse, record, ledgerPtr)
		page.Add(offerResponse)
	}

	page.FullURL = actions.FullURL(ctx)
	page.PopulateLinks()
	httpjson.Render(w, page, httpjson.HALJSON)
}
