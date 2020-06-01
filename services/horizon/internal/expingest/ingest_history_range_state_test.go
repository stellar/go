package expingest

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
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
		ctx:            context.Background(),
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		runner:         s.runner,
	}
	s.system.initMetrics()

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

	next, err := historyRangeState{fromLedger: 0, toLedger: 0}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 0]")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)

	next, err = historyRangeState{fromLedger: 0, toLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 100]")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)

	next, err = historyRangeState{fromLedger: 100, toLedger: 0}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 0]")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)

	next, err = historyRangeState{fromLedger: 100, toLedger: 99}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 99]")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *IngestHistoryRangeStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	next, err := historyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *IngestHistoryRangeStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	next, err := historyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *IngestHistoryRangeStateTestSuite) TestGetLatestLedgerReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), errors.New("my error")).Once()

	next, err := historyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "could not get latest history ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

// TestAnotherNodeIngested tests the case when another node has ingested the range.
// In such case we go back to `init` state without processing.
func (s *IngestHistoryRangeStateTestSuite) TestAnotherNodeIngested() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(200), nil).Once()

	next, err := historyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *IngestHistoryRangeStateTestSuite) TestRunTransactionProcessorsOnLedgerReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil).Once()

	s.runner.On("RunTransactionProcessorsOnLedger", uint32(100)).Return(io.StatsLedgerTransactionProcessorResults{}, errors.New("my error")).Once()

	next, err := historyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error processing ledger sequence=100: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *IngestHistoryRangeStateTestSuite) TestSuccess() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil).Once()

	for i := 100; i <= 200; i++ {
		s.runner.On("RunTransactionProcessorsOnLedger", uint32(i)).Return(io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	}

	s.historyQ.On("Commit").Return(nil).Once()

	next, err := historyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *IngestHistoryRangeStateTestSuite) TestSuccessOneLedger() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil).Once()

	s.runner.On("RunTransactionProcessorsOnLedger", uint32(100)).Return(io.StatsLedgerTransactionProcessorResults{}, nil).Once()

	s.historyQ.On("Commit").Return(nil).Once()

	next, err := historyRangeState{fromLedger: 100, toLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func TestReingestHistoryRangeStateTestSuite(t *testing.T) {
	suite.Run(t, new(ReingestHistoryRangeStateTestSuite))
}

type ReingestHistoryRangeStateTestSuite struct {
	suite.Suite
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	runner         *mockProcessorsRunner
	system         *System
}

func (s *ReingestHistoryRangeStateTestSuite) SetupTest() {
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.system = &System{
		ctx:            context.Background(),
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		runner:         s.runner,
	}

	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.historyQ.On("Begin").Return(nil).Once()
}

func (s *ReingestHistoryRangeStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.runner.AssertExpectations(t)
}

func (s *ReingestHistoryRangeStateTestSuite) TestInvalidRange() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil)

	err := s.system.ReingestRange(0, 0, false)
	s.Assert().EqualError(err, "invalid range: [0, 0]")

	err = s.system.ReingestRange(0, 100, false)
	s.Assert().EqualError(err, "invalid range: [0, 100]")

	err = s.system.ReingestRange(100, 0, false)
	s.Assert().EqualError(err, "invalid range: [100, 0]")

	err = s.system.ReingestRange(100, 99, false)
	s.Assert().EqualError(err, "invalid range: [100, 99]")
}

func (s *ReingestHistoryRangeStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil)
	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(0), nil).Once()

	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
}

func (s *ReingestHistoryRangeStateTestSuite) TestGetLastLedgerExpIngestNonBlockingError() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()

	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(0), errors.New("my error")).Once()

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
}

func (s *ReingestHistoryRangeStateTestSuite) TestReingestRangeOverlaps() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()

	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(190), nil).Once()

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().Equal(err, ErrReingestRangeConflict)
}

func (s *ReingestHistoryRangeStateTestSuite) TestReingestRangeOverlapsAtEnd() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()

	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(200), nil).Once()

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().Equal(err, ErrReingestRangeConflict)
}

func (s *ReingestHistoryRangeStateTestSuite) TestClearHistoryFails() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(0), nil).Once()

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
	toidFrom := toid.New(100, 0, 0)
	toidTo := toid.New(101, 0, 0)
	s.historyQ.On(
		"DeleteRangeAll", toidFrom.ToInt64(), toidTo.ToInt64(),
	).Return(errors.New("my error")).Once()

	s.historyQ.On("Rollback").Return(nil).Once()

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().EqualError(err, "error in DeleteRangeAll: my error")
}

func (s *ReingestHistoryRangeStateTestSuite) TestRunTransactionProcessorsOnLedgerReturnsError() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(0), nil).Once()

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
	toidFrom := toid.New(100, 0, 0)
	toidTo := toid.New(101, 0, 0)
	s.historyQ.On(
		"DeleteRangeAll", toidFrom.ToInt64(), toidTo.ToInt64(),
	).Return(nil).Once()

	s.runner.On("RunTransactionProcessorsOnLedger", uint32(100)).
		Return(io.StatsLedgerTransactionProcessorResults{}, errors.New("my error")).Once()
	s.historyQ.On("Rollback").Return(nil).Once()

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().EqualError(err, "error processing ledger sequence=100: my error")
}

func (s *ReingestHistoryRangeStateTestSuite) TestCommitFails() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(0), nil).Once()

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
	toidFrom := toid.New(100, 0, 0)
	toidTo := toid.New(101, 0, 0)
	s.historyQ.On(
		"DeleteRangeAll", toidFrom.ToInt64(), toidTo.ToInt64(),
	).Return(nil).Once()

	s.runner.On("RunTransactionProcessorsOnLedger", uint32(100)).Return(io.StatsLedgerTransactionProcessorResults{}, nil).Once()

	s.historyQ.On("Commit").Return(errors.New("my error")).Once()
	s.historyQ.On("Rollback").Return(nil).Once()

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().EqualError(err, "Error committing db transaction: my error")
}

func (s *ReingestHistoryRangeStateTestSuite) TestSuccess() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(0), nil).Once()

	for i := uint32(100); i <= uint32(200); i++ {
		s.historyQ.On("Begin").Return(nil).Once()
		s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()

		toidFrom := toid.New(int32(i), 0, 0)
		toidTo := toid.New(int32(i+1), 0, 0)
		s.historyQ.On(
			"DeleteRangeAll", toidFrom.ToInt64(), toidTo.ToInt64(),
		).Return(nil).Once()

		s.runner.On("RunTransactionProcessorsOnLedger", i).Return(io.StatsLedgerTransactionProcessorResults{}, nil).Once()

		s.historyQ.On("Commit").Return(nil).Once()
		s.historyQ.On("Rollback").Return(nil).Once()
	}

	err := s.system.ReingestRange(100, 200, false)
	s.Assert().NoError(err)
}

func (s *ReingestHistoryRangeStateTestSuite) TestSuccessOneLedger() {
	s.historyQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(0), nil).Once()

	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()

	toidFrom := toid.New(100, 0, 0)
	toidTo := toid.New(101, 0, 0)
	s.historyQ.On(
		"DeleteRangeAll", toidFrom.ToInt64(), toidTo.ToInt64(),
	).Return(nil).Once()

	s.runner.On("RunTransactionProcessorsOnLedger", uint32(100)).Return(io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	err := s.system.ReingestRange(100, 100, false)
	s.Assert().NoError(err)
}

func (s *ReingestHistoryRangeStateTestSuite) TestGetLastLedgerExpIngestError() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	err := s.system.ReingestRange(100, 200, true)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
}

func (s *ReingestHistoryRangeStateTestSuite) TestReingestRangeForce() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(190), nil).Once()

	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()

	toidFrom := toid.New(100, 0, 0)
	toidTo := toid.New(201, 0, 0)
	s.historyQ.On(
		"DeleteRangeAll", toidFrom.ToInt64(), toidTo.ToInt64(),
	).Return(nil).Once()

	for i := 100; i <= 200; i++ {
		s.runner.On("RunTransactionProcessorsOnLedger", uint32(i)).Return(io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	}

	s.historyQ.On("Commit").Return(nil).Once()

	err := s.system.ReingestRange(100, 200, true)
	s.Assert().NoError(err)
}
