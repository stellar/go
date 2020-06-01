package expingest

import (
	"context"
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
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
	ledgeBackend      *ledgerbackend.MockDatabaseBackend
	historyQ          *mockDBQ
	historyAdapter    *adapters.MockHistoryArchiveAdapter
	runner            *mockProcessorsRunner
	stellarCoreClient *mockStellarCoreClient
	system            *System
}

func (s *ResumeTestTestSuite) SetupTest() {
	s.ledgeBackend = &ledgerbackend.MockDatabaseBackend{}
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.stellarCoreClient = &mockStellarCoreClient{}
	s.system = &System{
		ctx:               context.Background(),
		historyQ:          s.historyQ,
		historyAdapter:    s.historyAdapter,
		runner:            s.runner,
		ledgerBackend:     s.ledgeBackend,
		stellarCoreClient: s.stellarCoreClient,
	}
	s.system.initMetrics()

	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *ResumeTestTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.runner.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.ledgeBackend.AssertExpectations(t)
	s.stellarCoreClient.AssertExpectations(t)
}

func (s *ResumeTestTestSuite) TestInvalidParam() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}

	next, err := resumeState{latestSuccessfullyProcessedLedger: 0}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "unexpected latestSuccessfullyProcessedLedger value")
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 100},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 100},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestGetLatestLedgerLessThanCurrent() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(99), nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "expected ingest ledger to be at most one greater than last ingested ledger in db")
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) TestGetIngestionVersionError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, errors.New("my error")).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting exp ingest version: my error")
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 100},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestIngestionVersionLessThanCurrentVersion() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) TestIngestionVersionGreaterThanCurrentVersion() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion+1, nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) TestGetLatestLedgerError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), errors.New("my error"))

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "could not get latest history ledger: my error")
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 100},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestLatestHistoryLedgerLessThanIngestLedger() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(99), nil)

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) TestLatestHistoryLedgerGreaterThanIngestLedger() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(101), nil)

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) mockSuccessfulIngestion() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(101), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(101), nil)

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	s.runner.On("RunAllProcessorsOnLedger", uint32(102)).Return(io.StatsChangeProcessorResults{}, io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(102)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(102),
	).Return(nil).Once()

	s.historyQ.On("GetExpStateInvalid").Return(false, nil).Once()
}
func (s *ResumeTestTestSuite) TestBumpIngestLedger() {
	s.mockSuccessfulIngestion()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 99}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 102},
			sleepDuration: 0,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestBumpIngestLedgerWhenIngestLedgerEqualsLastLedgerExpIngest() {
	s.mockSuccessfulIngestion()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 102},
			sleepDuration: 0,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestIngestAllMasterNode() {
	s.mockSuccessfulIngestion()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 101}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 102},
			sleepDuration: 0,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestErrorSettingCursorIgnored() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil)

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	s.runner.On("RunAllProcessorsOnLedger", uint32(101)).Return(io.StatsChangeProcessorResults{}, io.StatsLedgerTransactionProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(101)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(101),
	).Return(errors.New("my error")).Once()

	s.historyQ.On("GetExpStateInvalid").Return(false, nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 101},
			sleepDuration: 0,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) TestNoNewLedgers() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil)

	s.ledgeBackend.On("GetLatestLedgerSequence").Return(uint32(100), nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			// Check the same ledger later
			node: resumeState{latestSuccessfullyProcessedLedger: 100},
			// Sleep because we learned the ledger is not there yet.
			sleepDuration: defaultSleep,
		},
		next,
	)
}
