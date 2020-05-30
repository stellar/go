package history

import (
	"strconv"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

var (
	issuer            = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	twoEurOfferSeller = xdr.MustAddress("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")

	nativeAsset = xdr.MustNewNativeAsset()
	eurAsset    = xdr.MustNewCreditAsset("EUR", issuer.Address())
	usdAsset    = xdr.MustNewCreditAsset("USD", issuer.Address())

	eurOffer = xdr.OfferEntry{
		SellerId: issuer,
		OfferId:  xdr.Int64(4),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 1,
			D: 1,
		},
		Flags:  1,
		Amount: xdr.Int64(500),
	}
	twoEurOffer = xdr.OfferEntry{
		SellerId: twoEurOfferSeller,
		OfferId:  xdr.Int64(5),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 2,
			D: 1,
		},
		Flags:  2,
		Amount: xdr.Int64(500),
	}
	threeEurOffer = xdr.OfferEntry{
		SellerId: twoEurOfferSeller,
		OfferId:  xdr.Int64(50),
		Buying:   eurAsset,
		Selling:  nativeAsset,
		Price: xdr.Price{
			N: 3,
			D: 1,
		},
		Flags:  2,
		Amount: xdr.Int64(500),
	}
)

func assertOfferEntryMatchesDBOffer(t *testing.T, offerEntry xdr.OfferEntry, offer Offer, lastModifiedLedger xdr.Uint32) {
	if offerEntry.SellerId.Address() != offer.SellerID {
		t.Fatalf(
			"seller id in offer entry %v does not equal to seller id in offer from db %v",
			offerEntry.SellerId.Address(),
			offer.SellerID,
		)
	}
	if offerEntry.OfferId != offer.OfferID {
		t.Fatalf(
			"offer id in offer entry %v does not equal to offer id in offer from db %v",
			offerEntry.OfferId,
			offer.OfferID,
		)
	}
	if offerEntry.Selling.String() != offer.SellingAsset.String() {
		t.Fatalf(
			"selling asset in offer entry %v does not equal to selling asset in offer from db %v",
			offerEntry.Selling.String(),
			offer.SellingAsset.String(),
		)
	}
	if offerEntry.Buying.String() != offer.BuyingAsset.String() {
		t.Fatalf(
			"buying asset in offer entry %v does not equal to buying asset in offer from db %v",
			offerEntry.Buying.String(),
			offer.BuyingAsset.String(),
		)
	}
	if offerEntry.Amount != offer.Amount {
		t.Fatalf(
			"amount in offer entry %v does not equal to amount in offer from db %v",
			offerEntry.Amount,
			offer.Amount,
		)
	}
	if offerEntry.Price.N != xdr.Int32(offer.Pricen) {
		t.Fatalf(
			"price numerator in offer entry %v does not equal to price numerator in offer from db %v",
			offerEntry.Price.N,
			offer.Pricen,
		)
	}
	if offerEntry.Price.D != xdr.Int32(offer.Priced) {
		t.Fatalf(
			"price denominator in offer entry %v does not equal to price denominator in offer from db %v",
			offerEntry.Price.D,
			offer.Priced,
		)
	}
	if offerEntry.Flags != xdr.Uint32(offer.Flags) {
		t.Fatalf(
			"flags in offer entry %v does not equal to flags in offer from db %v",
			offerEntry.Flags,
			offer.Flags,
		)
	}
	if lastModifiedLedger != xdr.Uint32(offer.LastModifiedLedger) {
		t.Fatalf(
			"last_modified_ledger %v does not equal last_modified_ledger %v in offer from DB",
			lastModifiedLedger,
			offer,
		)
	}
}

func insertOffer(q *Q, offer xdr.OfferEntry, lastModifiedSeq uint32) error {
	batch := q.NewOffersBatchInsertBuilder(0)
	err := batch.Add(offer, xdr.Uint32(lastModifiedSeq))
	if err != nil {
		return err
	}
	return batch.Exec()
}

func TestGetOfferByID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := insertOffer(q, eurOffer, 1234)
	tt.Assert.NoError(err)
	offer, err := q.GetOfferByID(int64(eurOffer.OfferId))
	tt.Assert.NoError(err)
	assertOfferEntryMatchesDBOffer(t, eurOffer, offer, 1234)
}

func TestGetNonExistentOfferByID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	_, err := q.GetOfferByID(12345)
	tt.Assert.True(q.NoRows(err))
}

func TestQueryEmptyOffers(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	offers, err := q.GetAllOffers()
	tt.Assert.NoError(err)
	tt.Assert.Len(offers, 0)

	updated, err := q.GetUpdatedOffers(0)
	tt.Assert.NoError(err)
	tt.Assert.Len(updated, 0)

	count, err := q.CountOffers()
	tt.Assert.NoError(err)
	tt.Assert.Equal(0, count)

	numRemoved, err := q.CompactOffers(100)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), numRemoved)
	seq, err := q.GetOfferCompactionSequence()
	tt.Assert.NoError(err)
	tt.Assert.Equal(uint32(100), seq)
}

func TestInsertOffers(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := insertOffer(q, eurOffer, 1234)
	tt.Assert.NoError(err)
	err = insertOffer(q, twoEurOffer, 1235)
	tt.Assert.NoError(err)

	offers, err := q.GetAllOffers()
	tt.Assert.NoError(err)
	tt.Assert.Len(offers, 2)

	offersByID := map[xdr.Int64]Offer{
		offers[0].OfferID: offers[0],
		offers[1].OfferID: offers[1],
	}

	assertOfferEntryMatchesDBOffer(t, eurOffer, offersByID[eurOffer.OfferId], 1234)
	assertOfferEntryMatchesDBOffer(t, twoEurOffer, offersByID[twoEurOffer.OfferId], 1235)

	count, err := q.CountOffers()
	tt.Assert.NoError(err)
	tt.Assert.Equal(2, count)

	numRemoved, err := q.CompactOffers(12350)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), numRemoved)
	seq, err := q.GetOfferCompactionSequence()
	tt.Assert.NoError(err)
	tt.Assert.Equal(uint32(12350), seq)

	afterCompactionCount, err := q.CountOffers()
	tt.Assert.NoError(err)
	tt.Assert.Equal(2, afterCompactionCount)

	afterCompactionOffers, err := q.GetAllOffers()
	tt.Assert.NoError(err)
	tt.Assert.Len(afterCompactionOffers, 2)
}

func TestUpdateOffer(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := insertOffer(q, eurOffer, 1234)
	tt.Assert.NoError(err)

	offers, err := q.GetAllOffers()
	tt.Assert.NoError(err)
	tt.Assert.Len(offers, 1)

	updatedOffers, err := q.GetUpdatedOffers(1233)
	tt.Assert.NoError(err)
	tt.Assert.Equal(offers, updatedOffers)

	updatedOffers, err = q.GetUpdatedOffers(100)
	tt.Assert.NoError(err)
	tt.Assert.Equal(offers, updatedOffers)

	updatedOffers, err = q.GetUpdatedOffers(1234)
	tt.Assert.NoError(err)
	tt.Assert.Len(updatedOffers, 0)

	assertOfferEntryMatchesDBOffer(t, eurOffer, offers[0], 1234)

	modifiedEurOffer := eurOffer
	modifiedEurOffer.Amount -= 10

	rowsAffected, err := q.UpdateOffer(modifiedEurOffer, 1235)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsAffected)

	offers, err = q.GetAllOffers()
	tt.Assert.NoError(err)
	tt.Assert.Len(offers, 1)

	updatedOffers, err = q.GetUpdatedOffers(1234)
	tt.Assert.NoError(err)
	tt.Assert.Equal(offers, updatedOffers)

	updatedOffers, err = q.GetUpdatedOffers(1235)
	tt.Assert.NoError(err)
	tt.Assert.Len(updatedOffers, 0)

	assertOfferEntryMatchesDBOffer(t, modifiedEurOffer, offers[0], 1235)
}

func TestRemoveNonExistantOffer(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	numAffected, err := q.RemoveOffer(xdr.Int64(12345), 1236)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), numAffected)
}

func TestRemoveOffer(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := insertOffer(q, eurOffer, 1234)
	tt.Assert.NoError(err)
	offers, err := q.GetAllOffers()
	tt.Assert.NoError(err)
	tt.Assert.Len(offers, 1)
	assertOfferEntryMatchesDBOffer(t, eurOffer, offers[0], 1234)

	expectedUpdates := offers
	rowsAffected, err := q.RemoveOffer(eurOffer.OfferId, 1236)
	tt.Assert.Equal(int64(1), rowsAffected)
	tt.Assert.NoError(err)
	expectedUpdates[0].LastModifiedLedger = 1236
	expectedUpdates[0].Deleted = true

	offers, err = q.GetAllOffers()
	tt.Assert.NoError(err)
	tt.Assert.Len(offers, 0)

	offers, err = q.GetOffersByIDs([]int64{int64(expectedUpdates[0].OfferID)})
	tt.Assert.NoError(err)
	tt.Assert.Len(offers, 0)

	_, err = q.GetOfferByID(int64(expectedUpdates[0].OfferID))
	tt.Assert.True(q.NoRows(err))

	updated, err := q.GetUpdatedOffers(1234)
	tt.Assert.NoError(err)
	tt.Assert.Equal(expectedUpdates, updated)

	count, err := q.CompactOffers(1235)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), count)

	updated, err = q.GetUpdatedOffers(1234)
	tt.Assert.NoError(err)
	tt.Assert.Equal(expectedUpdates, updated)

	count, err = q.CompactOffers(1236)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), count)
	seq, err := q.GetOfferCompactionSequence()
	tt.Assert.NoError(err)
	tt.Assert.Equal(uint32(1236), seq)

	updated, err = q.GetUpdatedOffers(1234)
	tt.Assert.NoError(err)
	tt.Assert.Len(updated, 0)
}

func TestGetOffers(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	err := insertOffer(q, eurOffer, 1234)
	tt.Assert.NoError(err)
	err = insertOffer(q, twoEurOffer, 1235)
	tt.Assert.NoError(err)

	// check removed offers aren't included in GetOffer queries
	err = insertOffer(q, threeEurOffer, 1234)
	tt.Assert.NoError(err)
	count, err := q.RemoveOffer(threeEurOffer.OfferId, 1235)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), count)

	pageQuery, err := db2.NewPageQuery("", false, "", 10)
	tt.Assert.NoError(err)

	t.Run("Filter by selling asset", func(t *testing.T) {
		query := OffersQuery{
			PageQuery: pageQuery,
			Selling:   &usdAsset,
		}

		offers, err := q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 0)

		query = OffersQuery{
			PageQuery: pageQuery,
			Selling:   &nativeAsset,
		}

		offers, err = q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 2)

		for _, offer := range offers {
			tt.Assert.Equal(nativeAsset, offer.SellingAsset)
		}
	})

	t.Run("Filter by buying asset", func(t *testing.T) {
		query := OffersQuery{
			PageQuery: pageQuery,
			Buying:    &eurAsset,
		}

		offers, err := q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 2)

		for _, offer := range offers {
			tt.Assert.Equal(eurAsset, offer.BuyingAsset)
		}

		query = OffersQuery{
			PageQuery: pageQuery,
			Buying:    &usdAsset,
		}

		offers, err = q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 0)
	})

	t.Run("Filter by seller", func(t *testing.T) {
		sellerID := issuer.Address()
		query := OffersQuery{
			PageQuery: pageQuery,
			SellerID:  sellerID,
		}

		offers, err := q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 1)

		assertOfferEntryMatchesDBOffer(t, eurOffer, offers[0], 1234)
	})

	t.Run("PageQuery", func(t *testing.T) {
		pageQuery, err := db2.NewPageQuery("", false, "", 10)
		tt.Assert.NoError(err)

		query := OffersQuery{
			PageQuery: pageQuery,
		}

		offers, err := q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 2)

		offersByID := map[xdr.Int64]Offer{
			offers[0].OfferID: offers[0],
			offers[1].OfferID: offers[1],
		}

		assertOfferEntryMatchesDBOffer(t, eurOffer, offersByID[eurOffer.OfferId], 1234)
		assertOfferEntryMatchesDBOffer(t, twoEurOffer, offersByID[twoEurOffer.OfferId], 1235)

		pageQuery, err = db2.NewPageQuery("", false, "asc", 1)
		tt.Assert.NoError(err)
		query = OffersQuery{
			PageQuery: pageQuery,
		}

		offers, err = q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 1)

		assertOfferEntryMatchesDBOffer(t, eurOffer, offers[0], 1234)

		pageQuery, err = db2.NewPageQuery("", false, "desc", 1)
		tt.Assert.NoError(err)
		query = OffersQuery{
			PageQuery: pageQuery,
		}

		offers, err = q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 1)

		assertOfferEntryMatchesDBOffer(t, twoEurOffer, offers[0], 1235)

		pageQuery, err = db2.NewPageQuery(
			strconv.FormatInt(int64(eurOffer.OfferId), 10),
			false,
			"",
			10,
		)
		tt.Assert.NoError(err)
		query = OffersQuery{
			PageQuery: pageQuery,
		}

		offers, err = q.GetOffers(query)
		tt.Assert.NoError(err)
		tt.Assert.Len(offers, 1)

		assertOfferEntryMatchesDBOffer(t, twoEurOffer, offers[0], 1235)
	})
}
