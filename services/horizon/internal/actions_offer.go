package horizon

import (
	"errors"
	"fmt"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/render/hal"
)

// This file contains the actions:

// OffersByAccountAction renders a page of offer resources, for a given
// account.  These offers are present in the ledger as of the latest validated
// ledger.
type OffersByAccountAction struct {
	Action
	Address   string
	PageQuery db2.PageQuery
	Records   []core.Offer
	Ledgers   history.LedgerCache
	Page      hal.Page
}

// JSON is a method for actions.JSON
func (action *OffersByAccountAction) JSON() {
	action.Do(
		action.loadParams,
		action.loadRecords,
		action.loadLedgers,
		action.loadPage,
		func() {
			hal.Render(action.W, action.Page)
		},
	)
}

// SetupAndValidateSSE calls the setup functions before we can stream and validates
// the request parameters. Errors are stored in action.Err.
func (action *OffersByAccountAction) SetupAndValidateSSE() {
	action.Setup(
		action.loadParams,
		action.loadRecords,
		action.loadLedgers,
	)
}

// SSE is a method for actions.SSE that loads the latest offers by account and sends them to the stream.
func (action *OffersByAccountAction) SSE(stream sse.Stream) {
	// No point reloading data if Setup was just called.
	if action.InitialDataIsFresh == false {
		action.Do(
			action.loadParams,
			action.loadRecords,
			action.loadLedgers,
		)
	} else {
		action.InitialDataIsFresh = false
	}
	action.Do(func() {
		stream.SetLimit(int(action.PageQuery.Limit))
		for _, record := range action.Records[stream.SentCount():] {
			ledger, found := action.Ledgers.Records[record.Lastmodified]
			ledgerPtr := &ledger
			if !found {
				if action.App.config.AllowEmptyLedgerDataResponses {
					ledgerPtr = nil
				} else {
					msg := fmt.Sprintf("could not find ledger data for sequence %d", record.Lastmodified)
					stream.Err(errors.New(msg))
					return
				}
			}
			var res horizon.Offer
			resourceadapter.PopulateOffer(action.R.Context(), &res, record, ledgerPtr)
			stream.Send(sse.Event{ID: res.PagingToken(), Data: res})
		}
	},
	)
}

func (action *OffersByAccountAction) loadParams() {
	action.PageQuery = action.GetPageQuery()
	action.Address = action.GetString("account_id")
}

// loadLedgers populates the ledger cache for this action
func (action *OffersByAccountAction) loadLedgers() {
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
			if action.App.config.AllowEmptyLedgerDataResponses {
				ledgerPtr = nil
			} else {
				msg := fmt.Sprintf("could not find ledger data for sequence %d", record.Lastmodified)
				action.Err = errors.New(msg)
				return
			}
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
