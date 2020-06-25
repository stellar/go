package expingest

import (
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/suite"
)

func TestStressTestStateTestSuite(t *testing.T) {
	suite.Run(t, new(StressTestStateTestSuite))
}

type StressTestStateTestSuite struct {
	suite.Suite
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	runner         *mockProcessorsRunner
	system         *System
}

func (s *StressTestStateTestSuite) SetupTest() {
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.system = &System{
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		runner:         s.runner,
	}
	s.system.initMetrics()

	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.runner.On("EnableMemoryStatsLogging").Return()
	s.runner.On("SetLedgerBackend", fakeLedgerBackend{
		numTransactions:       10,
		changesPerTransaction: 4,
	}).Return()

}

func (s *StressTestStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.runner.AssertExpectations(t)
}

func (s *StressTestStateTestSuite) TestBounds() {
	*s.historyQ = mockDBQ{}
	*s.runner = mockProcessorsRunner{}

	err := s.system.StressTest(-1, 4)
	s.Assert().EqualError(err, "transactions must be positive")

	err = s.system.StressTest(0, 4)
	s.Assert().EqualError(err, "transactions must be positive")

	err = s.system.StressTest(100, -2)
	s.Assert().EqualError(err, "changes per transaction must be positive")
}

func (s *StressTestStateTestSuite) TestBeginReturnsError() {
	*s.historyQ = mockDBQ{}
	s.historyQ.On("GetTx").Return(nil).Once()
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	err := s.system.StressTest(10, 4)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
}

func (s *StressTestStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	err := s.system.StressTest(10, 4)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
}

func (s *StressTestStateTestSuite) TestGetLastLedgerExpIngestNonEmpty() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()

	err := s.system.StressTest(10, 4)
	s.Assert().EqualError(err, "Database not empty")
}

func (s *StressTestStateTestSuite) TestRunAllProcessorsOnLedgerReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunAllProcessorsOnLedger", uint32(1)).Return(io.StatsChangeProcessorResults{}, io.StatsLedgerTransactionProcessorResults{}, errors.New("my error")).Once()

	err := s.system.StressTest(10, 4)
	s.Assert().EqualError(err, "Error running processors on ledger: my error")
}

func (s *StressTestStateTestSuite) TestUpdateLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunAllProcessorsOnLedger", uint32(1)).Return(io.StatsChangeProcessorResults{}, io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(1)).Return(errors.New("my error")).Once()

	err := s.system.StressTest(10, 4)
	s.Assert().EqualError(err, "Error updating last ingested ledger: my error")
}

func (s *StressTestStateTestSuite) TestCommitReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunAllProcessorsOnLedger", uint32(1)).Return(io.StatsChangeProcessorResults{}, io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(1)).Return(nil).Once()
	s.historyQ.On("Commit").Return(errors.New("my error")).Once()

	err := s.system.StressTest(10, 4)
	s.Assert().EqualError(err, "Error committing db transaction: my error")
}

func (s *StressTestStateTestSuite) TestSucceeds() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunAllProcessorsOnLedger", uint32(1)).Return(io.StatsChangeProcessorResults{}, io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(1)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	err := s.system.StressTest(10, 4)
	s.Assert().NoError(err)
}
