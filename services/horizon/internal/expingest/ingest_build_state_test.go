package expingest

import (
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/suite"
)

func TestBuildStateTestSuite(t *testing.T) {
	suite.Run(t, new(BuildStateTestSuite))
}

type BuildStateTestSuite struct {
	suite.Suite
	graph          *orderbook.OrderBookGraph
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	system         *System
}

func (s *BuildStateTestSuite) SetupTest() {
	s.graph = orderbook.NewOrderBookGraph()
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.system = &System{
		state:          state{systemState: buildStateState, checkpointLedger: 63},
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		graph:          s.graph,
	}

	s.Assert().Equal(buildStateState, s.system.state.systemState)
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *BuildStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
}

func (s *BuildStateTestSuite) TestCheckPointLedgerIsZero() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()

	s.system.state.checkpointLedger = 0

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "unexpected checkpointLedger value")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *BuildStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *BuildStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(initState, nextState.systemState)
}
