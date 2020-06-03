package expingest

import (
	"fmt"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type IngestionStatusTestSuite struct {
	suite.Suite
	historyQ *mockDBQ
	stream   *OrderBookStream
}

func TestIngestionStatus(t *testing.T) {
	suite.Run(t, new(IngestionStatusTestSuite))
}

func (t *IngestionStatusTestSuite) SetupTest() {
	t.historyQ = &mockDBQ{}
	t.stream = NewOrderBookStream(t.historyQ, &mockOrderBookGraph{})
}

func (t *IngestionStatusTestSuite) TearDownTest() {
	t.historyQ.AssertExpectations(t.T())
}

func (t *IngestionStatusTestSuite) TestGetExpStateInvalidError() {
	t.historyQ.On("GetExpStateInvalid").
		Return(false, fmt.Errorf("state invalid error")).
		Once()
	_, err := t.stream.getIngestionStatus()
	t.Assert().EqualError(err, "Error from GetExpStateInvalid: state invalid error")
}

func (t *IngestionStatusTestSuite) TestGetLatestLedgerError() {
	t.historyQ.On("GetExpStateInvalid").
		Return(false, nil).
		Once()

	t.historyQ.On("GetLatestLedger").
		Return(uint32(0), fmt.Errorf("latest ledger error")).
		Once()
	_, err := t.stream.getIngestionStatus()
	t.Assert().EqualError(err, "Error from GetLatestLedger: latest ledger error")
}

func (t *IngestionStatusTestSuite) TestGetLastLedgerExpIngestNonBlockingError() {
	t.historyQ.On("GetExpStateInvalid").
		Return(false, nil).
		Once()

	t.historyQ.On("GetLatestLedger").
		Return(uint32(200), nil).
		Once()

	t.historyQ.On("GetLastLedgerExpIngestNonBlocking").
		Return(uint32(0), fmt.Errorf("exp ingest error")).
		Once()

	_, err := t.stream.getIngestionStatus()
	t.Assert().EqualError(err, "Error from GetLastLedgerExpIngestNonBlocking: exp ingest error")
}

func (t *IngestionStatusTestSuite) TestGetOfferCompactionSequenceError() {
	t.historyQ.On("GetExpStateInvalid").
		Return(false, nil).
		Once()

	t.historyQ.On("GetLatestLedger").
		Return(uint32(200), nil).
		Once()

	t.historyQ.On("GetLastLedgerExpIngestNonBlocking").
		Return(uint32(200), nil).
		Once()

	t.historyQ.On("GetOfferCompactionSequence").
		Return(uint32(0), fmt.Errorf("compaction error")).
		Once()

	_, err := t.stream.getIngestionStatus()
	t.Assert().EqualError(err, "Error from GetOfferCompactionSequence: compaction error")
}

func (t *IngestionStatusTestSuite) TestStateInvalid() {
	t.historyQ.On("GetExpStateInvalid").
		Return(true, nil).
		Once()

	t.historyQ.On("GetLatestLedger").
		Return(uint32(200), nil).
		Once()

	t.historyQ.On("GetLastLedgerExpIngestNonBlocking").
		Return(uint32(200), nil).
		Once()

	t.historyQ.On("GetOfferCompactionSequence").
		Return(uint32(100), nil).
		Once()

	status, err := t.stream.getIngestionStatus()
	t.Assert().NoError(err)
	t.Assert().Equal(ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               true,
		LastIngestedLedger:         200,
		LastOfferCompactionLedger:  100,
	}, status)
}

func (t *IngestionStatusTestSuite) TestHistoryInconsistentWithState() {
	t.historyQ.On("GetExpStateInvalid").
		Return(true, nil).
		Once()

	t.historyQ.On("GetLatestLedger").
		Return(uint32(200), nil).
		Once()

	t.historyQ.On("GetLastLedgerExpIngestNonBlocking").
		Return(uint32(201), nil).
		Once()

	t.historyQ.On("GetOfferCompactionSequence").
		Return(uint32(100), nil).
		Once()

	status, err := t.stream.getIngestionStatus()
	t.Assert().NoError(err)
	t.Assert().Equal(ingestionStatus{
		HistoryConsistentWithState: false,
		StateInvalid:               true,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}, status)
}

func (t *IngestionStatusTestSuite) TestHistoryLatestLedgerZero() {
	t.historyQ.On("GetExpStateInvalid").
		Return(false, nil).
		Once()

	t.historyQ.On("GetLatestLedger").
		Return(uint32(0), nil).
		Once()

	t.historyQ.On("GetLastLedgerExpIngestNonBlocking").
		Return(uint32(201), nil).
		Once()

	t.historyQ.On("GetOfferCompactionSequence").
		Return(uint32(100), nil).
		Once()

	status, err := t.stream.getIngestionStatus()
	t.Assert().NoError(err)
	t.Assert().Equal(ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}, status)
}

type UpdateOrderBookStreamTestSuite struct {
	suite.Suite
	historyQ *mockDBQ
	graph    *mockOrderBookGraph
	stream   *OrderBookStream
}

func TestUpdateOrderBookStream(t *testing.T) {
	suite.Run(t, new(UpdateOrderBookStreamTestSuite))
}

func (t *UpdateOrderBookStreamTestSuite) SetupTest() {
	t.historyQ = &mockDBQ{}
	t.graph = &mockOrderBookGraph{}
	t.stream = NewOrderBookStream(t.historyQ, t.graph)
}

func (t *UpdateOrderBookStreamTestSuite) TearDownTest() {
	t.historyQ.AssertExpectations(t.T())
	t.graph.AssertExpectations(t.T())
}

func (t *UpdateOrderBookStreamTestSuite) TestGetAllOffersError() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.graph.On("Clear").Return().Once()
	t.graph.On("Discard").Return().Once()
	t.historyQ.On("GetAllOffers").
		Return([]history.Offer{}, fmt.Errorf("offers error")).
		Once()

	t.stream.lastLedger = 300
	_, err := t.stream.update(status)
	t.Assert().EqualError(err, "Error from GetAllOffers: offers error")
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) TestResetApplyError() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.graph.On("Clear").Return().Once()
	t.graph.On("Discard").Return().Once()

	sellerID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	offer := history.Offer{OfferID: 1, SellerID: sellerID}
	offerEntry := xdr.OfferEntry{
		SellerId: xdr.MustAddress(sellerID),
		OfferId:  1,
	}
	otherOffer := history.Offer{OfferID: 20, SellerID: sellerID}
	otherOfferEntry := xdr.OfferEntry{
		SellerId: xdr.MustAddress(sellerID),
		OfferId:  20,
	}
	t.historyQ.On("GetAllOffers").
		Return([]history.Offer{offer, otherOffer}, nil).
		Once()

	t.graph.On("AddOffer", offerEntry).Return().Once()
	t.graph.On("AddOffer", otherOfferEntry).Return().Once()

	t.graph.On("Apply", status.LastIngestedLedger).
		Return(fmt.Errorf("apply error")).
		Once()

	t.stream.lastLedger = 300
	_, err := t.stream.update(status)
	t.Assert().EqualError(err, "Error applying changes to order book: apply error")
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) mockReset(status ingestionStatus) {
	t.graph.On("Clear").Return().Once()
	t.graph.On("Discard").Return().Once()

	sellerID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	offer := history.Offer{OfferID: 1, SellerID: sellerID}
	offerEntry := xdr.OfferEntry{
		SellerId: xdr.MustAddress(sellerID),
		OfferId:  1,
	}
	otherOffer := history.Offer{OfferID: 20, SellerID: sellerID}
	otherOfferEntry := xdr.OfferEntry{
		SellerId: xdr.MustAddress(sellerID),
		OfferId:  20,
	}
	offers := []history.Offer{offer, otherOffer}
	t.historyQ.On("GetAllOffers").
		Return(offers, nil).
		Once()

	t.graph.On("AddOffer", offerEntry).Return().Once()
	t.graph.On("AddOffer", otherOfferEntry).Return().Once()

	t.graph.On("Apply", status.LastIngestedLedger).
		Return(nil).
		Once()
}

func (t *UpdateOrderBookStreamTestSuite) TestFirstUpdateSucceeds() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.mockReset(status)

	reset, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
	t.Assert().True(reset)
}

func (t *UpdateOrderBookStreamTestSuite) TestInvalidState() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               true,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.graph.On("Clear").Return().Once()

	reset, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
	t.Assert().True(reset)

	t.stream.lastLedger = 123

	t.graph.On("Clear").Return().Once()

	reset, err = t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
	t.Assert().True(reset)
}

func (t *UpdateOrderBookStreamTestSuite) TestHistoryInconsistentWithState() {
	status := ingestionStatus{
		HistoryConsistentWithState: false,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.graph.On("Clear").Return().Once()

	reset, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
	t.Assert().True(reset)

	t.stream.lastLedger = 123

	t.graph.On("Clear").Return().Once()

	reset, err = t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
	t.Assert().True(reset)
}

func (t *UpdateOrderBookStreamTestSuite) TestLastIngestedLedgerBehindStream() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.mockReset(status)

	t.stream.lastLedger = 300
	reset, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
	t.Assert().True(reset)
}

func (t *UpdateOrderBookStreamTestSuite) TestStreamBehindLastCompactionLedger() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.mockReset(status)

	t.stream.lastLedger = 99
	reset, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
	t.Assert().True(reset)
}

func (t *UpdateOrderBookStreamTestSuite) TestStreamLedgerEqualsLastIngestedLedger() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}

	t.stream.lastLedger = 201
	reset, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
	t.Assert().False(reset)
}

func (t *UpdateOrderBookStreamTestSuite) TestGetUpdatedOffersError() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.graph.On("Discard").Return().Once()

	t.stream.lastLedger = 100
	t.historyQ.MockQOffers.On("GetUpdatedOffers", uint32(100)).
		Return([]history.Offer{}, fmt.Errorf("updated offers error")).
		Once()

	_, err := t.stream.update(status)
	t.Assert().EqualError(err, "Error from GetUpdatedOffers: updated offers error")
	t.Assert().Equal(uint32(100), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) mockUpdate() {
	t.stream.lastLedger = 100

	t.graph.On("Discard").Return().Once()
	sellerID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	offer := history.Offer{OfferID: 1, SellerID: sellerID, LastModifiedLedger: 101}
	offerEntry := xdr.OfferEntry{
		SellerId: xdr.MustAddress(sellerID),
		OfferId:  1,
	}
	otherOffer := history.Offer{OfferID: 20, SellerID: sellerID, LastModifiedLedger: 102}
	otherOfferEntry := xdr.OfferEntry{
		SellerId: xdr.MustAddress(sellerID),
		OfferId:  20,
	}
	deletedOffer := history.Offer{OfferID: 30, SellerID: sellerID, LastModifiedLedger: 103, Deleted: true}
	offers := []history.Offer{offer, otherOffer, deletedOffer}
	t.historyQ.MockQOffers.On("GetUpdatedOffers", t.stream.lastLedger).
		Return(offers, nil).
		Once()

	t.graph.On("AddOffer", offerEntry).Return().Once()
	t.graph.On("AddOffer", otherOfferEntry).Return().Once()
	t.graph.On("RemoveOffer", deletedOffer.OfferID).Return(t.graph).Once()
}

func (t *UpdateOrderBookStreamTestSuite) TestApplyUpdatesError() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}

	t.mockUpdate()

	t.graph.On("Apply", status.LastIngestedLedger).
		Return(fmt.Errorf("apply error")).
		Once()

	_, err := t.stream.update(status)
	t.Assert().EqualError(err, "Error applying changes to order book: apply error")
	t.Assert().Equal(uint32(100), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) TestApplyUpdatesSucceeds() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}

	t.mockUpdate()

	t.graph.On("Apply", status.LastIngestedLedger).
		Return(nil).
		Once()

	reset, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(status.LastIngestedLedger, t.stream.lastLedger)
	t.Assert().False(reset)
}

type VerifyOrderBookStreamTestSuite struct {
	suite.Suite
	historyQ    *mockDBQ
	graph       *mockOrderBookGraph
	stream      *OrderBookStream
	initialTime time.Time
}

func TestVerifyOrderBookStream(t *testing.T) {
	suite.Run(t, new(VerifyOrderBookStreamTestSuite))
}

func (t *VerifyOrderBookStreamTestSuite) SetupTest() {
	t.historyQ = &mockDBQ{}
	t.graph = &mockOrderBookGraph{}
	t.stream = NewOrderBookStream(t.historyQ, t.graph)
	t.initialTime = t.stream.lastVerification

	sellerID := "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"
	otherSellerID := "GAXI33UCLQTCKM2NMRBS7XYBR535LLEVAHL5YBN4FTCB4HZHT7ZA5CVK"
	t.graph.On("Offers").Return([]xdr.OfferEntry{
		{
			SellerId: xdr.MustAddress(sellerID),
			OfferId:  1,
			Selling:  xdr.MustNewNativeAsset(),
			Buying:   xdr.MustNewCreditAsset("USD", sellerID),
			Amount:   123,
			Price: xdr.Price{
				N: 1,
				D: 2,
			},
			Flags: 1,
			Ext:   xdr.OfferEntryExt{},
		},
		{
			SellerId: xdr.MustAddress(otherSellerID),
			OfferId:  3,
			Selling:  xdr.MustNewCreditAsset("EUR", sellerID),
			Buying:   xdr.MustNewCreditAsset("CHF", sellerID),
			Amount:   9,
			Price: xdr.Price{
				N: 3,
				D: 1,
			},
			Flags: 0,
			Ext:   xdr.OfferEntryExt{},
		},
	}).Once()
}

func (t *VerifyOrderBookStreamTestSuite) TearDownTest() {
	t.historyQ.AssertExpectations(t.T())
	t.graph.AssertExpectations(t.T())
}

func (t *VerifyOrderBookStreamTestSuite) TestGetAllOffersError() {
	t.historyQ.On("GetAllOffers").
		Return([]history.Offer{}, fmt.Errorf("offers error")).
		Once()

	t.stream.lastLedger = 300
	t.stream.verifyAllOffers()
	t.Assert().Equal(uint32(300), t.stream.lastLedger)
	t.Assert().True(t.stream.lastVerification.Equal(t.initialTime))
}

func (t *VerifyOrderBookStreamTestSuite) TestEmptyDBOffers() {
	var offers []history.Offer
	t.historyQ.On("GetAllOffers").Return(offers, nil).Once()

	t.stream.lastLedger = 300
	t.stream.verifyAllOffers()
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
	t.Assert().False(t.stream.lastVerification.Equal(t.initialTime))
}

func (t *VerifyOrderBookStreamTestSuite) TestLengthMismatch() {
	offers := []history.Offer{
		{
			OfferID:            1,
			SellerID:           "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			SellingAsset:       xdr.MustNewNativeAsset(),
			BuyingAsset:        xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			Amount:             123,
			Pricen:             1,
			Priced:             2,
			Price:              0.5,
			Flags:              1,
			Deleted:            false,
			LastModifiedLedger: 1,
		},
	}
	t.historyQ.On("GetAllOffers").Return(offers, nil).Once()

	t.stream.lastLedger = 300
	t.stream.verifyAllOffers()
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
	t.Assert().False(t.stream.lastVerification.Equal(t.initialTime))
}

func (t *VerifyOrderBookStreamTestSuite) TestContentMismatch() {
	offers := []history.Offer{
		{
			OfferID:            1,
			SellerID:           "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			SellingAsset:       xdr.MustNewNativeAsset(),
			BuyingAsset:        xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			Amount:             123,
			Pricen:             1,
			Priced:             2,
			Price:              0.5,
			Flags:              1,
			Deleted:            false,
			LastModifiedLedger: 1,
		},
		{
			OfferID:            3,
			SellerID:           "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			SellingAsset:       xdr.MustNewNativeAsset(),
			BuyingAsset:        xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			Amount:             123,
			Pricen:             1,
			Priced:             2,
			Price:              0.5,
			Flags:              1,
			Deleted:            false,
			LastModifiedLedger: 1,
		},
	}
	t.historyQ.On("GetAllOffers").Return(offers, nil).Once()

	t.stream.lastLedger = 300
	t.stream.verifyAllOffers()
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
	t.Assert().False(t.stream.lastVerification.Equal(t.initialTime))
}

func (t *VerifyOrderBookStreamTestSuite) TestSuccess() {
	offers := []history.Offer{
		{
			OfferID:            1,
			SellerID:           "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			SellingAsset:       xdr.MustNewNativeAsset(),
			BuyingAsset:        xdr.MustNewCreditAsset("USD", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			Amount:             123,
			Pricen:             1,
			Priced:             2,
			Price:              0.5,
			Flags:              1,
			Deleted:            false,
			LastModifiedLedger: 1,
		},
		{
			OfferID:            3,
			SellerID:           "GAXI33UCLQTCKM2NMRBS7XYBR535LLEVAHL5YBN4FTCB4HZHT7ZA5CVK",
			SellingAsset:       xdr.MustNewCreditAsset("EUR", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			BuyingAsset:        xdr.MustNewCreditAsset("CHF", "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			Amount:             9,
			Pricen:             3,
			Priced:             1,
			Price:              3,
			Flags:              0,
			Deleted:            false,
			LastModifiedLedger: 1,
		},
	}
	t.historyQ.On("GetAllOffers").Return(offers, nil).Once()

	t.stream.lastLedger = 300
	t.stream.verifyAllOffers()
	t.Assert().Equal(uint32(300), t.stream.lastLedger)
	t.Assert().False(t.stream.lastVerification.Equal(t.initialTime))
}
