package expingest

import (
	"database/sql"
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestVerifyRangeStateTestSuite(t *testing.T) {
	suite.Run(t, new(VerifyRangeStateTestSuite))
}

type VerifyRangeStateTestSuite struct {
	suite.Suite
	graph          *mockOrderBookGraph
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	runner         *mockProcessorsRunner
	system         *System
}

func (s *VerifyRangeStateTestSuite) SetupTest() {
	s.graph = &mockOrderBookGraph{}
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.system = &System{
		state: state{
			systemState: verifyRangeState,
		},
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		runner:         s.runner,
		graph:          s.graph,
	}

	s.Assert().Equal(verifyRangeState, s.system.state.systemState)
	// Checking if in tx in runCurrentState()
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.graph.On("Discard").Once()
}

func (s *VerifyRangeStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.runner.AssertExpectations(t)
	s.graph.AssertExpectations(t)
}

func (s *VerifyRangeStateTestSuite) TestInvalidRange() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	*s.graph = mockOrderBookGraph{}
	s.historyQ.On("GetTx").Return(nil).Maybe()

	s.system.state.rangeFromLedger = 0
	s.system.state.rangeToLedger = 0
	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 0]")
	s.Assert().Equal(shutdownState, nextState.systemState)

	s.system.state.rangeFromLedger = 0
	s.system.state.rangeToLedger = 100
	nextState, err = s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 100]")
	s.Assert().Equal(shutdownState, nextState.systemState)

	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 0
	nextState, err = s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 0]")
	s.Assert().Equal(shutdownState, nextState.systemState)

	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 99
	nextState, err = s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 99]")
	s.Assert().Equal(shutdownState, nextState.systemState)
}

func (s *VerifyRangeStateTestSuite) TestBeginReturnsError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	*s.graph = mockOrderBookGraph{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(shutdownState, nextState.systemState)
}

func (s *VerifyRangeStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(shutdownState, nextState.systemState)
}

func (s *VerifyRangeStateTestSuite) TestGetLastLedgerExpIngestNonEmpty() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Database not empty")
	s.Assert().Equal(shutdownState, nextState.systemState)
}

func (s *VerifyRangeStateTestSuite) TestRunHistoryArchiveIngestionReturnsError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()

	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error ingesting history archive: my error")
	s.Assert().Equal(shutdownState, nextState.systemState)
}

func (s *VerifyRangeStateTestSuite) TestSuccess() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(100)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()
	s.graph.On("Apply", uint32(s.system.state.rangeFromLedger)).Return(nil).Once()

	for i := uint32(101); i <= 200; i++ {
		s.historyQ.On("Begin").Return(nil).Once()
		s.runner.On("RunAllProcessorsOnLedger", i).Return(nil).Once()
		s.historyQ.On("UpdateLastLedgerExpIngest", i).Return(nil).Once()
		s.historyQ.On("Commit").Return(nil).Once()
		s.graph.On("Apply", i).Return(nil).Once()
	}

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(shutdownState, nextState.systemState)
}

func (s *VerifyRangeStateTestSuite) TestSuccessWithVerify() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 110
	s.system.state.rangeVerifyState = true

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(100)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()
	s.graph.On("Apply", uint32(s.system.state.rangeFromLedger)).Return(nil).Once()

	for i := uint32(101); i <= 110; i++ {
		s.historyQ.On("Begin").Return(nil).Once()
		s.runner.On("RunAllProcessorsOnLedger", i).Return(nil).Once()
		s.historyQ.On("UpdateLastLedgerExpIngest", i).Return(nil).Once()
		s.historyQ.On("Commit").Return(nil).Once()
		s.graph.On("Apply", i).Return(nil).Once()
	}

	s.graph.On("OffersMap").Return(map[xdr.Int64]xdr.OfferEntry{
		eurOffer.OfferId: xdr.OfferEntry{
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
		},
		twoEurOffer.OfferId: xdr.OfferEntry{
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
		},
	}).Once()

	clonedQ := &mockDBQ{}
	s.historyQ.On("CloneIngestionQ").Return(clonedQ).Once()

	clonedQ.On("BeginTx", mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*sql.TxOptions)
		s.Assert().Equal(sql.LevelRepeatableRead, arg.Isolation)
		s.Assert().True(arg.ReadOnly)
	}).Return(errors.New("my error")).Once()
	clonedQ.On("Rollback").Return(nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting transaction: my error")
	s.Assert().Equal(shutdownState, nextState.systemState)
	clonedQ.AssertExpectations(s.T())
}
