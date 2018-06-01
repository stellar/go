package horizon

import (
	"net/http"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/support/render/hal"
)

// OrderBookShowAction renders a account summary found by its address.
type OrderBookShowAction struct {
	Action
	Selling  xdr.Asset
	Buying   xdr.Asset
	Record   core.OrderBookSummary
	Resource horizon.OrderBookSummary
	Limit    uint64
}

// LoadQuery sets action.Query from the request params
func (action *OrderBookShowAction) LoadQuery() {
	action.Selling = action.GetAsset("selling_")
	action.Buying = action.GetAsset("buying_")
	action.Limit = action.GetLimit("limit", 20, 200)

	if action.Err != nil {
		action.Err = &problem.P{
			Type:   "invalid_order_book",
			Title:  "Invalid Order Book Parameters",
			Status: http.StatusBadRequest,
			Detail: "The parameters that specify what order book to view are invalid in some way. " +
				"Please ensure that your type parameters (selling_asset_type and buying_asset_type) are one the " +
				"following valid values: native, credit_alphanum4, credit_alphanum12.  Also ensure that you " +
				"have specified selling_asset_code and selling_asset_issuer if selling_asset_type is not 'native', as well " +
				"as buying_asset_code and buying_asset_issuer if buying_asset_type is not 'native'",
		}
	}
}

// LoadRecord populates action.Record
func (action *OrderBookShowAction) LoadRecord() {
	action.Err = action.CoreQ().GetOrderBookSummary(
		&action.Record,
		action.Selling,
		action.Buying,
		action.Limit,
	)
}

// LoadResource populates action.Record
func (action *OrderBookShowAction) LoadResource() {
	action.Err = resourceadapter.PopulateOrderBookSummary(
		action.R.Context(),
		&action.Resource,
		action.Selling,
		action.Buying,
		action.Record,
	)
}

// JSON is a method for actions.JSON
func (action *OrderBookShowAction) JSON() {
	action.Do(action.LoadQuery, action.LoadRecord, action.LoadResource)

	action.Do(func() {
		hal.Render(action.W, action.Resource)
	})
}

// SSE is a method for actions.SSE
func (action *OrderBookShowAction) SSE(stream sse.Stream) {
	action.Do(action.LoadQuery, action.LoadRecord, action.LoadResource)

	action.Do(func() {
		stream.SetLimit(10)
		stream.Send(sse.Event{
			Data: action.Resource,
		})
	})

}
