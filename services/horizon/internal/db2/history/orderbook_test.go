package history

import (
	"database/sql"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestGetOrderBookSummaryRequiresTransaction(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.GetOrderBookSummary(nativeAsset, eurAsset, 10)
	assert.EqualError(t, err, "cannot be called outside of a transaction")

	assert.NoError(t, q.Begin())
	defer q.Rollback()

	_, err = q.GetOrderBookSummary(nativeAsset, eurAsset, 10)
	assert.EqualError(t, err, "should only be called in a repeatable read transaction")
}

func TestGetOrderBookSummary(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	asksButNoBids := []xdr.OfferEntry{twoEurOffer}
	asksButNoBidsResponse := OrderBookSummary{
		Asks: []PriceLevel{
			{
				Pricen: int32(twoEurOffer.Price.N),
				Priced: int32(twoEurOffer.Price.D),
				Pricef: "2.0000000",
				Amount: "0.0000500",
			},
		},
	}

	sellEurOffer := twoEurOffer
	sellEurOffer.Buying, sellEurOffer.Selling = sellEurOffer.Selling, sellEurOffer.Buying
	sellEurOffer.OfferId = 15
	bidsButNoAsks := []xdr.OfferEntry{sellEurOffer}
	bidsButNoAsksResponse := OrderBookSummary{
		Bids: []PriceLevel{
			{
				Pricen: int32(sellEurOffer.Price.D),
				Priced: int32(sellEurOffer.Price.N),
				Pricef: "0.5000000",
				Amount: "0.0000500",
			},
		},
	}

	otherEurOffer := twoEurOffer
	otherEurOffer.Amount = xdr.Int64(math.MaxInt64)
	otherEurOffer.OfferId = 16

	nonCanonicalPriceTwoEurOffer := twoEurOffer
	nonCanonicalPriceTwoEurOffer.OfferId = 30
	// Add a separate offer with the same price value, but
	// using a non-canonical representation, to make sure
	// they are coalesced into the same price level
	nonCanonicalPriceTwoEurOffer.Price.N *= 15
	nonCanonicalPriceTwoEurOffer.Price.D *= 15

	sellEurOffer.Price.N = 9
	sellEurOffer.Price.D = 10

	otherSellEurOffer := sellEurOffer
	otherSellEurOffer.OfferId = 17
	// sellEurOffer.Price * 2
	otherSellEurOffer.Price.N = 9
	otherSellEurOffer.Price.D = 5

	fullOffers := []xdr.OfferEntry{
		twoEurOffer,
		otherEurOffer,
		nonCanonicalPriceTwoEurOffer,
		threeEurOffer,
		sellEurOffer,
		otherSellEurOffer,
	}

	fullResponse := OrderBookSummary{
		Asks: []PriceLevel{
			{
				Pricen: int32(twoEurOffer.Price.N),
				Priced: int32(twoEurOffer.Price.D),
				Pricef: "2.0000000",
				Amount: "922337203685.4776807",
			},
			{
				Pricen: int32(threeEurOffer.Price.N),
				Priced: int32(threeEurOffer.Price.D),
				Pricef: "3.0000000",
				Amount: "0.0000500",
			},
		},
		Bids: []PriceLevel{
			{
				Pricen: int32(sellEurOffer.Price.D),
				Priced: int32(sellEurOffer.Price.N),
				Pricef: "1.1111111",
				Amount: "0.0000500",
			},
			{
				Pricen: int32(otherSellEurOffer.Price.D),
				Priced: int32(otherSellEurOffer.Price.N),
				Pricef: "0.5555556",
				Amount: "0.0000500",
			},
		},
	}

	limitResponse := OrderBookSummary{
		Asks: []PriceLevel{
			{
				Pricen: int32(twoEurOffer.Price.N),
				Priced: int32(twoEurOffer.Price.D),
				Pricef: "2.0000000",
				Amount: "922337203685.4776807",
			},
		},
		Bids: []PriceLevel{
			{
				Pricen: int32(sellEurOffer.Price.D),
				Priced: int32(sellEurOffer.Price.N),
				Pricef: "1.1111111",
				Amount: "0.0000500",
			},
		},
	}

	for _, testCase := range []struct {
		name     string
		offers   []xdr.OfferEntry
		limit    int
		expected OrderBookSummary
	}{
		{
			"empty orderbook",
			[]xdr.OfferEntry{},
			10,
			OrderBookSummary{},
		},
		{
			"orderbook with asks but no bids",
			asksButNoBids,
			10,
			asksButNoBidsResponse,
		},
		{
			"orderbook with bids but no asks",
			bidsButNoAsks,
			10,
			bidsButNoAsksResponse,
		},
		{
			"full orderbook",
			fullOffers,
			10,
			fullResponse,
		},
		{
			"limit summaries",
			fullOffers,
			1,
			limitResponse,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.NoError(t, q.TruncateTables([]string{"offers"}))

			batch := q.NewOffersBatchInsertBuilder(0)
			for i, offer := range testCase.offers {
				assert.NoError(t, batch.Add(offer, xdr.Uint32(i+1)))
			}
			assert.NoError(t, batch.Exec())

			assert.NoError(t, q.BeginTx(&sql.TxOptions{
				Isolation: sql.LevelRepeatableRead,
				ReadOnly:  true,
			}))
			defer q.Rollback()

			result, err := q.GetOrderBookSummary(nativeAsset, eurAsset, testCase.limit)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestGetOrderBookSummaryExcludesRemovedOffers(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	sellEurOffer := twoEurOffer
	sellEurOffer.Buying, sellEurOffer.Selling = sellEurOffer.Selling, sellEurOffer.Buying
	sellEurOffer.OfferId = 15
	offers := []xdr.OfferEntry{
		twoEurOffer,
		threeEurOffer,
		sellEurOffer,
	}

	batch := q.NewOffersBatchInsertBuilder(0)
	for i, offer := range offers {
		assert.NoError(t, batch.Add(offer, xdr.Uint32(i+1)))
	}
	assert.NoError(t, batch.Exec())

	assert.NoError(t, q.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}))

	result, err := q.GetOrderBookSummary(nativeAsset, eurAsset, 100)
	assert.NoError(t, err)
	assert.Len(t, result.Asks, 2)
	assert.Len(t, result.Bids, 1)

	assert.NoError(t, q.Rollback())

	for i, offer := range offers {
		var count int64
		count, err = q.RemoveOffer(offer.OfferId, uint32(i+2))
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	}

	assert.NoError(t, q.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}))

	result, err = q.GetOrderBookSummary(nativeAsset, eurAsset, 100)
	assert.NoError(t, err)
	assert.Len(t, result.Asks, 0)
	assert.Len(t, result.Bids, 0)

	assert.NoError(t, q.Rollback())

	count, err := q.CompactOffers(1000)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(offers)), count)

	assert.NoError(t, q.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}))

	result, err = q.GetOrderBookSummary(nativeAsset, eurAsset, 100)
	assert.NoError(t, err)
	assert.Len(t, result.Asks, 0)
	assert.Len(t, result.Bids, 0)

	assert.NoError(t, q.Rollback())
}
