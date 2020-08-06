package history

import (
	"database/sql"
	"math"
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
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

	asksButNoBids := []xdr.LedgerEntry{twoEurOffer}
	asksButNoBidsResponse := OrderBookSummary{
		Asks: []PriceLevel{
			{
				Pricen: int32(twoEurOffer.Data.Offer.Price.N),
				Priced: int32(twoEurOffer.Data.Offer.Price.D),
				Pricef: "2.0000000",
				Amount: "0.0000500",
			},
		},
	}

	sellEurOffer := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.OfferEntry{
				SellerId: twoEurOfferSeller,
				OfferId:  xdr.Int64(15),
				Buying:   nativeAsset,
				Selling:  eurAsset,
				Price: xdr.Price{
					N: 2,
					D: 1,
				},
				Flags:  2,
				Amount: xdr.Int64(500),
			},
		},
	}

	bidsButNoAsks := []xdr.LedgerEntry{sellEurOffer}
	bidsButNoAsksResponse := OrderBookSummary{
		Bids: []PriceLevel{
			{
				Pricen: int32(sellEurOffer.Data.Offer.Price.D),
				Priced: int32(sellEurOffer.Data.Offer.Price.N),
				Pricef: "0.5000000",
				Amount: "0.0000500",
			},
		},
	}

	otherEurOffer := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.OfferEntry{
				SellerId: twoEurOfferSeller,
				OfferId:  xdr.Int64(6),
				Buying:   eurAsset,
				Selling:  nativeAsset,
				Price: xdr.Price{
					N: 2,
					D: 1,
				},
				Flags:  2,
				Amount: xdr.Int64(math.MaxInt64),
			},
		},
	}

	nonCanonicalPriceTwoEurOffer := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.OfferEntry{
				SellerId: twoEurOfferSeller,
				OfferId:  xdr.Int64(30),
				Buying:   eurAsset,
				Selling:  nativeAsset,
				Price: xdr.Price{
					// Add a separate offer with the same price value, but
					// using a non-canonical representation, to make sure
					// they are coalesced into the same price level
					N: 2 * 15,
					D: 1 * 15,
				},
				Flags:  2,
				Amount: xdr.Int64(500),
			},
		},
	}

	otherSellEurOffer := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.OfferEntry{
				SellerId: twoEurOfferSeller,
				OfferId:  xdr.Int64(17),
				Buying:   nativeAsset,
				Selling:  eurAsset,
				Price: xdr.Price{
					N: 9,
					D: 5,
				},
				Flags:  2,
				Amount: xdr.Int64(500),
			},
		},
	}

	fullOffers := []xdr.LedgerEntry{
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
				Pricen: int32(twoEurOffer.Data.Offer.Price.N),
				Priced: int32(twoEurOffer.Data.Offer.Price.D),
				Pricef: "2.0000000",
				Amount: "922337203685.4776807",
			},
			{
				Pricen: int32(threeEurOffer.Data.Offer.Price.N),
				Priced: int32(threeEurOffer.Data.Offer.Price.D),
				Pricef: "3.0000000",
				Amount: "0.0000500",
			},
		},
		Bids: []PriceLevel{
			{
				Pricen: int32(otherSellEurOffer.Data.Offer.Price.D),
				Priced: int32(otherSellEurOffer.Data.Offer.Price.N),
				Pricef: "0.5555556",
				Amount: "0.0000500",
			},
			{
				Pricen: int32(sellEurOffer.Data.Offer.Price.D),
				Priced: int32(sellEurOffer.Data.Offer.Price.N),
				Pricef: "0.5000000",
				Amount: "0.0000500",
			},
		},
	}

	limitResponse := OrderBookSummary{
		Asks: []PriceLevel{
			{
				Pricen: int32(twoEurOffer.Data.Offer.Price.N),
				Priced: int32(twoEurOffer.Data.Offer.Price.D),
				Pricef: "2.0000000",
				Amount: "922337203685.4776807",
			},
		},
		Bids: []PriceLevel{
			{
				Pricen: int32(otherSellEurOffer.Data.Offer.Price.D),
				Priced: int32(otherSellEurOffer.Data.Offer.Price.N),
				Pricef: "0.5555556",
				Amount: "0.0000500",
			},
		},
	}

	for _, testCase := range []struct {
		name     string
		offers   []xdr.LedgerEntry
		limit    int
		expected OrderBookSummary
	}{
		{
			"empty orderbook",
			[]xdr.LedgerEntry{},
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
			for _, offer := range testCase.offers {
				assert.NoError(t, batch.Add(offer))
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

	sellEurOffer := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.OfferEntry{
				SellerId: twoEurOfferSeller,
				OfferId:  xdr.Int64(15),
				Buying:   nativeAsset,
				Selling:  eurAsset,
				Price: xdr.Price{
					N: 2,
					D: 1,
				},
				Flags:  2,
				Amount: xdr.Int64(500),
			},
		},
	}

	offers := []xdr.LedgerEntry{
		twoEurOffer,
		threeEurOffer,
		sellEurOffer,
	}

	batch := q.NewOffersBatchInsertBuilder(0)
	for _, offer := range offers {
		assert.NoError(t, batch.Add(offer))
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
		count, err = q.RemoveOffer(offer.Data.Offer.OfferId, uint32(i+2))
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
