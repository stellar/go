package expingest

import (
	"context"
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestResumeTestTestSuite(t *testing.T) {
	suite.Run(t, new(ResumeTestTestSuite))
}

type ResumeTestTestSuite struct {
	suite.Suite
	graph             *mockOrderBookGraph
	ledgeBackend      *ledgerbackend.MockDatabaseBackend
	historyQ          *mockDBQ
	historyAdapter    *adapters.MockHistoryArchiveAdapter
	runner            *mockProcessorsRunner
	stellarCoreClient *mockStellarCoreClient
	system            *System
}

func (s *ResumeTestTestSuite) SetupTest() {
	s.graph = &mockOrderBookGraph{}
	s.ledgeBackend = &ledgerbackend.MockDatabaseBackend{}
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.stellarCoreClient = &mockStellarCoreClient{}
	s.system = &System{
		ctx: context.Background(),
		state: state{
			systemState:                       resumeState,
			latestSuccessfullyProcessedLedger: 100,
		},
		historyQ:          s.historyQ,
		historyAdapter:    s.historyAdapter,
		runner:            s.runner,
		ledgerBackend:     s.ledgeBackend,
		graph:             s.graph,
		stellarCoreClient: s.stellarCoreClient,
	}

	s.Assert().Equal(resumeState, s.system.state.systemState)
	// Checking if in tx in runCurrentState()
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.graph.On("Discard").Once()
}

func (s *ResumeTestTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.runner.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.ledgeBackend.AssertExpectations(t)
	s.stellarCoreClient.AssertExpectations(t)
	s.graph.AssertExpectations(t)
}

func (s *ResumeTestTestSuite) TestInvalidParam() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Maybe()

	// Recreate orderbook graph to remove Discard assertion
	*s.graph = mockOrderBookGraph{}

	s.system.state.latestSuccessfullyProcessedLedger = 0
	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "unexpected latestSuccessfullyProcessedLedger value")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *ResumeTestTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	// Recreate orderbook graph to remove Discard assertion
	*s.graph = mockOrderBookGraph{}

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(resumeState, nextState.systemState)
	s.Assert().Equal(uint32(100), nextState.latestSuccessfullyProcessedLedger)
}

func (s *ResumeTestTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(resumeState, nextState.systemState)
	s.Assert().Equal(uint32(100), nextState.latestSuccessfullyProcessedLedger)
}

func (s *ResumeTestTestSuite) TestGetLatestLedgerLessThanCurrent() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(99), nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "expected ingest ledger to be at most one greater than last ingested ledger in db")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *ResumeTestTestSuite) TestIngestOrderbookOnly() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(110), nil).Once()

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	// Rollback to release the lock as we're not updating DB
	s.historyQ.On("Rollback").Return(nil).Once()
	s.runner.On("RunOrderBookProcessorOnLedger", uint32(101)).Return(nil).Once()
	s.graph.On("Apply", uint32(101)).Return(nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(resumeState, nextState.systemState)
	s.Assert().Equal(uint32(101), nextState.latestSuccessfullyProcessedLedger)
	s.Assert().True(nextState.noSleep)
}

// TestIngestOrderbookOnlyWhenLastLedgerExpEqualsCurrent is very similar to the test above
// but it checks the `ingestLedger <= lastIngestedLedger` that, if incorrect, could lead
// to a nasty off-by-one error.
func (s *ResumeTestTestSuite) TestIngestOrderbookOnlyWhenLastLedgerExpEqualsCurrent() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(101), nil).Once()

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	// Rollback to release the lock as we're not updating DB
	s.historyQ.On("Rollback").Return(nil).Once()
	s.runner.On("RunOrderBookProcessorOnLedger", uint32(101)).Return(nil).Once()
	s.graph.On("Apply", uint32(101)).Return(nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(resumeState, nextState.systemState)
	s.Assert().Equal(uint32(101), nextState.latestSuccessfullyProcessedLedger)
	s.Assert().True(nextState.noSleep)
}

func (s *ResumeTestTestSuite) TestIngestAllMasterNode() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	s.runner.On("RunAllProcessorsOnLedger", uint32(101)).Return(nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(101)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()
	s.graph.On("Apply", uint32(101)).Return(nil).Once()

	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(101),
	).Return(nil).Once()

	s.historyQ.On("GetExpStateInvalid").Return(false, nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(resumeState, nextState.systemState)
	s.Assert().Equal(uint32(101), nextState.latestSuccessfullyProcessedLedger)
	s.Assert().True(nextState.noSleep)
}

func (s *ResumeTestTestSuite) TestErrorSettingCursorIgnored() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	s.runner.On("RunAllProcessorsOnLedger", uint32(101)).Return(nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(101)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()
	s.graph.On("Apply", uint32(101)).Return(nil).Once()

	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(101),
	).Return(errors.New("my error")).Once()

	s.historyQ.On("GetExpStateInvalid").Return(false, nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(resumeState, nextState.systemState)
	s.Assert().Equal(uint32(101), nextState.latestSuccessfullyProcessedLedger)
	s.Assert().True(nextState.noSleep)
}

func (s *ResumeTestTestSuite) TestNoNewLedgers() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(100), nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(resumeState, nextState.systemState)
	// Check the same ledger later
	s.Assert().Equal(uint32(100), nextState.latestSuccessfullyProcessedLedger)
	// Sleep because we learned the ledger is not there yet.
	s.Assert().False(nextState.noSleep)
}
