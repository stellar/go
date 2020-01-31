package expingest

import (
	"context"
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/suite"
)

func TestIngestHistoryRangeStateTestSuite(t *testing.T) {
	suite.Run(t, new(IngestHistoryRangeStateTestSuite))
}

type IngestHistoryRangeStateTestSuite struct {
	suite.Suite
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	runner         *mockProcessorsRunner
	system         *System
}

func (s *IngestHistoryRangeStateTestSuite) SetupTest() {
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.system = &System{
		ctx: context.Background(),
		state: state{
			systemState: ingestHistoryRangeState,
		},
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		runner:         s.runner,
	}

	s.Assert().Equal(ingestHistoryRangeState, s.system.state.systemState)
	// Checking if in tx in runCurrentState()
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *IngestHistoryRangeStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.runner.AssertExpectations(t)
}

func (s *IngestHistoryRangeStateTestSuite) TestInvalidRange() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Maybe()

	s.system.state.rangeFromLedger = 0
	s.system.state.rangeToLedger = 0
	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 0]")
	s.Assert().Equal(initState, nextState.systemState)

	s.system.state.rangeFromLedger = 0
	s.system.state.rangeToLedger = 100
	nextState, err = s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 100]")
	s.Assert().Equal(initState, nextState.systemState)

	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 0
	nextState, err = s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 0]")
	s.Assert().Equal(initState, nextState.systemState)

	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 99
	nextState, err = s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 99]")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestBeginReturnsError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestGetLatestLedgerReturnsError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "could not get latest history ledger: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

// TestAnotherNodeIngested tests the case when another node has ingested the range.
// In such case we go back to `init` state without processing.
func (s *IngestHistoryRangeStateTestSuite) TestAnotherNodeIngested() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(200), nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestRunTransactionProcessorsOnLedgerReturnsError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil).Once()

	s.runner.On("RunTransactionProcessorsOnLedger", uint32(100)).Return(errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error processing ledger sequence=100: my error")
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestSuccess() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil).Once()

	for i := 100; i <= 200; i++ {
		s.runner.On("RunTransactionProcessorsOnLedger", uint32(i)).Return(nil).Once()
	}

	s.historyQ.On("Commit").Return(nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestSuccessOneLedger() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 100

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil).Once()

	s.runner.On("RunTransactionProcessorsOnLedger", uint32(100)).Return(nil).Once()

	s.historyQ.On("Commit").Return(nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(initState, nextState.systemState)
}

// TestClearHistorySuccess tests clearing history before ingesting history range.
func (s *IngestHistoryRangeStateTestSuite) TestClearHistorySuccess() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200
	s.system.state.rangeClearHistory = true

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil).Once()

	toidFrom := int64(429496729600)
	toidTo := int64(863288426496)
	s.historyQ.On("DeleteRangeAll", toidFrom, toidTo).Return(nil).Once()

	from := toid.Parse(toidFrom)
	s.Assert().Equal(int32(100), from.LedgerSequence)
	s.Assert().Equal(int32(0), from.TransactionOrder)
	s.Assert().Equal(int32(0), from.OperationOrder)

	to := toid.Parse(toidTo)
	s.Assert().Equal(int32(201), to.LedgerSequence)
	s.Assert().Equal(int32(0), to.TransactionOrder)
	s.Assert().Equal(int32(0), to.OperationOrder)

	for i := 100; i <= 200; i++ {
		s.runner.On("RunTransactionProcessorsOnLedger", uint32(i)).Return(nil).Once()
	}

	s.historyQ.On("Commit").Return(nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(initState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestShutdownWhenDoneSuccess() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200
	s.system.state.shutdownWhenDone = true

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()

	for i := 100; i <= 200; i++ {
		s.runner.On("RunTransactionProcessorsOnLedger", uint32(i)).Return(nil).Once()
	}

	s.historyQ.On("Commit").Return(nil).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().NoError(err)
	s.Assert().Equal(shutdownState, nextState.systemState)
}

func (s *IngestHistoryRangeStateTestSuite) TestShutdownWhenDoneWithError() {
	s.system.state.rangeFromLedger = 100
	s.system.state.rangeToLedger = 200
	s.system.state.shutdownWhenDone = true

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()

	for i := 100; i <= 200; i++ {
		s.runner.On("RunTransactionProcessorsOnLedger", uint32(i)).Return(nil).Once()
	}

	s.historyQ.On("Commit").Return(errors.New("my error")).Once()

	nextState, err := s.system.runCurrentState()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error committing db transaction: my error")
	s.Assert().Equal(shutdownState, nextState.systemState)
}
