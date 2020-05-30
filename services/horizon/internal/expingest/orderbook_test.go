package expingest

import (
	"fmt"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
	"testing"
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
	t.stream = &OrderBookStream{HistoryQ: t.historyQ}
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
	t.stream = &OrderBookStream{OrderBookGraph: t.graph, HistoryQ: t.historyQ}
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
	_, _, err := t.stream.update(status)
	t.Assert().EqualError(err, "Error from loadOffersIntoGraph: GetAllOffers error: offers error")
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
	_, _, err := t.stream.update(status)
	t.Assert().EqualError(err, "Error applying changes to order book: apply error")
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) mockReset(status ingestionStatus) []history.Offer {
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
	return offers
}

func (t *UpdateOrderBookStreamTestSuite) TestFirstUpdateSucceeds() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	offers := t.mockReset(status)

	updated, removed, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Equal(offers, updated)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) TestInvalidState() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               true,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.graph.On("Clear").Return().Once()

	updated, removed, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Empty(updated)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)

	t.stream.lastLedger = 123

	t.graph.On("Clear").Return().Once()

	updated, removed, err = t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Empty(updated)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) TestHistoryInconsistentWithState() {
	status := ingestionStatus{
		HistoryConsistentWithState: false,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	t.graph.On("Clear").Return().Once()

	updated, removed, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Empty(updated)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)

	t.stream.lastLedger = 123

	t.graph.On("Clear").Return().Once()

	updated, removed, err = t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Empty(updated)
	t.Assert().Equal(uint32(0), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) TestLastIngestedLedgerBehindStream() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	offers := t.mockReset(status)

	t.stream.lastLedger = 300
	updated, removed, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Equal(offers, updated)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) TestStreamBehindLastCompactionLedger() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}
	offers := t.mockReset(status)

	t.stream.lastLedger = 99
	updated, removed, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Equal(offers, updated)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) TestStreamLedgerEqualsLastIngestedLedger() {
	status := ingestionStatus{
		HistoryConsistentWithState: true,
		StateInvalid:               false,
		LastIngestedLedger:         201,
		LastOfferCompactionLedger:  100,
	}

	t.stream.lastLedger = 201
	updated, removed, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Empty(removed)
	t.Assert().Empty(updated)
	t.Assert().Equal(uint32(201), t.stream.lastLedger)
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

	_, _, err := t.stream.update(status)
	t.Assert().EqualError(err, "Error from GetUpdatedOffers: updated offers error")
	t.Assert().Equal(uint32(100), t.stream.lastLedger)
}

func (t *UpdateOrderBookStreamTestSuite) mockUpdate() ([]history.Offer, []xdr.Int64) {
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

	return offers[:2], []xdr.Int64{deletedOffer.OfferID}
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

	_, _, err := t.stream.update(status)
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

	expectedUpdates, expectedRemoved := t.mockUpdate()

	t.graph.On("Apply", status.LastIngestedLedger).
		Return(nil).
		Once()

	updates, removed, err := t.stream.update(status)
	t.Assert().NoError(err)
	t.Assert().Equal(status.LastIngestedLedger, t.stream.lastLedger)
	t.Assert().Equal(expectedUpdates, updates)
	t.Assert().Equal(expectedRemoved, removed)
}
