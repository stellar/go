package actions

import (
	"context"
	"math/big"
	"net/http"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/exp/orderbook"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// StreamableObjectResponse is an interface for objects returned by streamable object endpoints
// A streamable object endpoint is an SSE endpoint which returns a single JSON object response
// instead of a page of items.
type StreamableObjectResponse interface {
	Equals(other StreamableObjectResponse) bool
}

// OrderBookResponse is the response for the /order_book endpoint
// OrderBookResponse implements StreamableObjectResponse
type OrderBookResponse struct {
	protocol.OrderBookSummary
}

func priceLevelsEqual(a, b []protocol.PriceLevel) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// Equals returns true if the OrderBookResponse is equal to `other`
func (o OrderBookResponse) Equals(other StreamableObjectResponse) bool {
	otherOrderBook, ok := other.(OrderBookResponse)
	if !ok {
		return false
	}
	return otherOrderBook.Selling == o.Selling &&
		otherOrderBook.Buying == o.Buying &&
		priceLevelsEqual(otherOrderBook.Bids, o.Bids) &&
		priceLevelsEqual(otherOrderBook.Asks, o.Asks)
}

var invalidOrderBook = problem.P{
	Type:   "invalid_order_book",
	Title:  "Invalid Order Book Parameters",
	Status: http.StatusBadRequest,
	Detail: "The parameters that specify what order book to view are invalid in some way. " +
		"Please ensure that your type parameters (selling_asset_type and buying_asset_type) are one the " +
		"following valid values: native, credit_alphanum4, credit_alphanum12.  Also ensure that you " +
		"have specified selling_asset_code and selling_asset_issuer if selling_asset_type is not 'native', as well " +
		"as buying_asset_code and buying_asset_issuer if buying_asset_type is not 'native'",
}

// GetOrderbookHandler is the action handler for the /order_book endpoint
type GetOrderbookHandler struct {
	OrderBookGraph *orderbook.OrderBookGraph
}

func offersToPriceLevels(offers []xdr.OfferEntry, invert bool) ([]protocol.PriceLevel, error) {
	result := []protocol.PriceLevel{}

	amountForPrice := map[xdr.Price]*big.Int{}
	for _, offer := range offers {
		offerAmount := big.NewInt(int64(offer.Amount))
		if amount, ok := amountForPrice[offer.Price]; ok {
			amount.Add(amount, offerAmount)
		} else {
			amountForPrice[offer.Price] = offerAmount
		}
	}
	for _, offer := range offers {
		total, ok := amountForPrice[offer.Price]
		if !ok {
			continue
		}
		delete(amountForPrice, offer.Price)

		offerPrice := offer.Price
		if invert {
			offerPrice.Invert()
		}

		amountString, err := amount.IntStringToAmount(total.String())
		if err != nil {
			return nil, err
		}

		result = append(result, protocol.PriceLevel{
			PriceR: protocol.Price{
				N: int32(offerPrice.N),
				D: int32(offerPrice.D),
			},
			Price:  offerPrice.String(),
			Amount: amountString,
		})
	}

	return result, nil
}

func (handler GetOrderbookHandler) orderBookSummary(
	ctx context.Context, selling, buying xdr.Asset, limit int,
) (protocol.OrderBookSummary, error) {
	response := protocol.OrderBookSummary{}
	if err := resourceadapter.PopulateAsset(ctx, &response.Selling, selling); err != nil {
		return response, err
	}
	if err := resourceadapter.PopulateAsset(ctx, &response.Buying, buying); err != nil {
		return response, err
	}

	var err error
	asks, bids := handler.OrderBookGraph.FindAsksAndBids(selling, buying, limit)
	if response.Asks, err = offersToPriceLevels(asks, false); err != nil {
		return response, err
	}

	if response.Bids, err = offersToPriceLevels(bids, true); err != nil {
		return response, err
	}

	return response, nil
}

// GetResource implements the /order_book endpoint
func (handler GetOrderbookHandler) GetResource(r *http.Request) (StreamableObjectResponse, error) {
	selling, err := GetAsset(r, "selling_")
	if err != nil {
		return nil, invalidOrderBook
	}
	buying, err := GetAsset(r, "buying_")
	if err != nil {
		return nil, invalidOrderBook
	}
	limit, err := GetLimit(r, "limit", 20, 200)
	if err != nil {
		return nil, invalidOrderBook
	}

	summary, err := handler.orderBookSummary(r.Context(), selling, buying, int(limit))
	if err != nil {
		return nil, err
	}

	return OrderBookResponse{summary}, nil
}
