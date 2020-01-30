package expingest

import (
	"sort"
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/orderbook"
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
	graph          *orderbook.OrderBookGraph
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	system         *System
}

func (s *InitStateTestSuite) SetupTest() {
	s.graph = orderbook.NewOrderBookGraph()
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.system = &System{
		state:          state{systemState: initState},
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		graph:          s.graph,
	}

	s.Assert().Equal(initState, s.system.state.systemState)
	// Checking if in tx in runCurrentState()
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *InitStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
}

func (s *InitStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *InitStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *InitStateTestSuite) TestGetExpIngestVersionReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting exp ingest version: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *InitStateTestSuite) TestGetLatestLedgerReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last history ledger sequence: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *InitStateTestSuite) TestBuildStateEmptyDatabase() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(buildStateState, nextState.systemState)
	s.Assert().Equal(uint32(63), nextState.checkpointLedger)
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

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(waitForCheckpointState, nextState.systemState)
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

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(ingestHistoryRangeState, nextState.systemState)
	s.Assert().Equal(uint32(101), nextState.rangeFromLedger)
	s.Assert().Equal(uint32(127), nextState.rangeToLedger)
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

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(buildStateState, nextState.systemState)
	s.Assert().Equal(uint32(127), nextState.checkpointLedger)
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

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(initState, nextState.systemState)
}

// TestResumeStateBehind is testing the case when:
// * state doesn't need to be rebuilt,
// * history is behind of expingest.
func (s *InitStateTestSuite) TestResumeStateBehind() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(130), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(ingestHistoryRangeState, nextState.systemState)
	s.Assert().Equal(uint32(101), nextState.rangeFromLedger)
	s.Assert().Equal(uint32(130), nextState.rangeToLedger)
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

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(resumeState, nextState.systemState)
	s.Assert().Equal(uint32(130), nextState.latestSuccessfullyProcessedLedger)

	// Also make sure offers are loaded
	offers := s.graph.Offers()
	sort.Slice(offers, func(i, j int) bool {
		return offers[i].OfferId < offers[j].OfferId
	})
	s.Assert().Equal([]xdr.OfferEntry{eurOffer, twoEurOffer}, offers)
}
