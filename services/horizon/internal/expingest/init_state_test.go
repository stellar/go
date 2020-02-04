package expingest

import (
	"context"
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestInitStateTestSuite(t *testing.T) {
	suite.Run(t, new(InitStateTestSuite))
}

type InitStateTestSuite struct {
	suite.Suite
	graph          *mockOrderBookGraph
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	system         *System
}

func (s *InitStateTestSuite) SetupTest() {
	s.graph = &mockOrderBookGraph{}
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.system = &System{
		ctx:            context.Background(),
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		graph:          s.graph,
	}

	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *InitStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.graph.AssertExpectations(t)
}

func (s *InitStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestGetExpIngestVersionReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting exp ingest version: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestCurrentVersionIsOutdated() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(1), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion+1, nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: stopState{}, sleepDuration: 0}, next)
}

func (s *InitStateTestSuite) TestGetLatestLedgerReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last history ledger sequence: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestBuildStateEmptyDatabase() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: buildState{checkpointLedger: 63}, sleepDuration: defaultSleep},
		next,
	)
}

// TestBuildStateWait is testing the case when:
// * the ingest system version has been incremented or no expingest ledger,
// * the old system is in front of the latest checkpoint.
func (s *InitStateTestSuite) TestBuildStateWait() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: waitForCheckpointState{}, sleepDuration: 0},
		next,
	)
}

// TestBuildStateCatchup is testing the case when:
// * the ingest system version has been incremented or no expingest ledger,
// * the old system is behind the latest checkpoint.
func (s *InitStateTestSuite) TestBuildStateCatchup() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(127), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          historyRangeState{fromLedger: 101, toLedger: 127},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestBuildStateOldHistory is testing the case when:
// * the ingest system version has been incremented or no expingest ledger,
// * the old system latest ledger is equal to the latest checkpoint.
func (s *InitStateTestSuite) TestBuildStateOldHistory() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(127), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(127), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(127), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          buildState{checkpointLedger: 127},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestResumeStateBehind is testing the case when:
// * state doesn't need to be rebuilt,
// * history is in front of expingest.
func (s *InitStateTestSuite) TestResumeStateInFront() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(130), nil).Once()

	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

// TestResumeStateBehind is testing the case when:
// * state doesn't need to be rebuilt,
// * history is behind of expingest.
func (s *InitStateTestSuite) TestResumeStateBehind() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(130), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          historyRangeState{fromLedger: 101, toLedger: 130},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestResumeStateBehindHistory0 is testing the case when:
// * state doesn't need to be rebuilt or was just rebuilt,
// * there are no ledgers in history tables.
// In such case we load offers and continue ingesting the next ledger.
func (s *InitStateTestSuite) TestResumeStateBehindHistory0() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(130), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

	s.historyQ.On("GetAllOffers").Return(
		[]history.Offer{
			history.Offer{
				OfferID:      eurOffer.OfferId,
				SellerID:     eurOffer.SellerId.Address(),
				SellingAsset: eurOffer.Selling,
				BuyingAsset:  eurOffer.Buying,
				Amount:       eurOffer.Amount,
				Pricen:       int32(eurOffer.Price.N),
				Priced:       int32(eurOffer.Price.D),
				Price:        float64(eurOffer.Price.N) / float64(eurOffer.Price.D),
				Flags:        uint32(eurOffer.Flags),
			},
			history.Offer{
				OfferID:      twoEurOffer.OfferId,
				SellerID:     twoEurOffer.SellerId.Address(),
				SellingAsset: twoEurOffer.Selling,
				BuyingAsset:  twoEurOffer.Buying,
				Amount:       twoEurOffer.Amount,
				Pricen:       int32(twoEurOffer.Price.N),
				Priced:       int32(twoEurOffer.Price.D),
				Price:        float64(twoEurOffer.Price.N) / float64(twoEurOffer.Price.D),
				Flags:        uint32(twoEurOffer.Flags),
			},
		},
		nil,
	).Once()

	s.graph.On("Clear").Once()
	s.graph.On("AddOffer", xdr.OfferEntry{
		SellerId: eurOffer.SellerId,
		OfferId:  eurOffer.OfferId,
		Selling:  eurOffer.Selling,
		Buying:   eurOffer.Buying,
		Amount:   eurOffer.Amount,
		Price: xdr.Price{
			N: xdr.Int32(eurOffer.Price.N),
			D: xdr.Int32(eurOffer.Price.D),
		},
		Flags: xdr.Uint32(eurOffer.Flags),
	}).Once()
	s.graph.On("AddOffer", xdr.OfferEntry{
		SellerId: twoEurOffer.SellerId,
		OfferId:  twoEurOffer.OfferId,
		Selling:  twoEurOffer.Selling,
		Buying:   twoEurOffer.Buying,
		Amount:   twoEurOffer.Amount,
		Price: xdr.Price{
			N: xdr.Int32(twoEurOffer.Price.N),
			D: xdr.Int32(twoEurOffer.Price.D),
		},
		Flags: xdr.Uint32(twoEurOffer.Flags),
	}).Once()
	s.graph.On("Apply", uint32(130)).Return(nil).Once()
	s.graph.On("Discard").Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 130},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestResumeStateBehind is testing the case when:
// * state doesn't need to be rebuilt,
// * history is in sync with expingest.
func (s *InitStateTestSuite) TestResumeStateSync() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(130), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(130), nil).Once()

	s.historyQ.On("GetAllOffers").Return(
		[]history.Offer{
			history.Offer{
				OfferID:      eurOffer.OfferId,
				SellerID:     eurOffer.SellerId.Address(),
				SellingAsset: eurOffer.Selling,
				BuyingAsset:  eurOffer.Buying,
				Amount:       eurOffer.Amount,
				Pricen:       int32(eurOffer.Price.N),
				Priced:       int32(eurOffer.Price.D),
				Price:        float64(eurOffer.Price.N) / float64(eurOffer.Price.D),
				Flags:        uint32(eurOffer.Flags),
			},
			history.Offer{
				OfferID:      twoEurOffer.OfferId,
				SellerID:     twoEurOffer.SellerId.Address(),
				SellingAsset: twoEurOffer.Selling,
				BuyingAsset:  twoEurOffer.Buying,
				Amount:       twoEurOffer.Amount,
				Pricen:       int32(twoEurOffer.Price.N),
				Priced:       int32(twoEurOffer.Price.D),
				Price:        float64(twoEurOffer.Price.N) / float64(twoEurOffer.Price.D),
				Flags:        uint32(twoEurOffer.Flags),
			},
		},
		nil,
	).Once()

	s.graph.On("Clear").Once()
	s.graph.On("AddOffer", xdr.OfferEntry{
		SellerId: eurOffer.SellerId,
		OfferId:  eurOffer.OfferId,
		Selling:  eurOffer.Selling,
		Buying:   eurOffer.Buying,
		Amount:   eurOffer.Amount,
		Price: xdr.Price{
			N: xdr.Int32(eurOffer.Price.N),
			D: xdr.Int32(eurOffer.Price.D),
		},
		Flags: xdr.Uint32(eurOffer.Flags),
	}).Once()
	s.graph.On("AddOffer", xdr.OfferEntry{
		SellerId: twoEurOffer.SellerId,
		OfferId:  twoEurOffer.OfferId,
		Selling:  twoEurOffer.Selling,
		Buying:   twoEurOffer.Buying,
		Amount:   twoEurOffer.Amount,
		Price: xdr.Price{
			N: xdr.Int32(twoEurOffer.Price.N),
			D: xdr.Int32(twoEurOffer.Price.D),
		},
		Flags: xdr.Uint32(twoEurOffer.Flags),
	}).Once()
	s.graph.On("Apply", uint32(130)).Return(nil).Once()
	s.graph.On("Discard").Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 130},
			sleepDuration: defaultSleep,
		},
		next,
	)
}
