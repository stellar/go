//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package ingest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func TestResumeTestTestSuite(t *testing.T) {
	suite.Run(t, new(ResumeTestTestSuite))
}

type ResumeTestTestSuite struct {
	suite.Suite
	ledgerBackend     *ledgerbackend.MockDatabaseBackend
	historyQ          *mockDBQ
	historyAdapter    *mockHistoryArchiveAdapter
	runner            *mockProcessorsRunner
	stellarCoreClient *mockStellarCoreClient
	system            *system
}

func (s *ResumeTestTestSuite) SetupTest() {
	s.ledgerBackend = &ledgerbackend.MockDatabaseBackend{}
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &mockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.stellarCoreClient = &mockStellarCoreClient{}
	s.system = &system{
		ctx:               context.Background(),
		historyQ:          s.historyQ,
		historyAdapter:    s.historyAdapter,
		runner:            s.runner,
		ledgerBackend:     s.ledgerBackend,
		stellarCoreClient: s.stellarCoreClient,
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}
	s.system.initMetrics()

	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *ResumeTestTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.runner.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.ledgerBackend.AssertExpectations(t)
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

func (s *ResumeTestTestSuite) TestRangeNotPreparedFailPrepare() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(101), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(101), nil)

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(102)).Return(false, nil).Once()
	s.ledgerBackend.On("PrepareRange", ledgerbackend.UnboundedRange(102)).Return(errors.New("my error")).Once()
	// Rollback twice (first one mocked in SetupTest) because we want to release
	// a distributed ingestion lock.
	s.historyQ.On("Rollback").Return(nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error preparing range: my error")
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) TestRangeNotPreparedSuccessPrepare() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(101), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(101), nil)

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(102)).Return(false, nil).Once()
	s.ledgerBackend.On("PrepareRange", ledgerbackend.UnboundedRange(102)).Return(nil).Once()
	// Rollback twice (first one mocked in SetupTest) because we want to release
	// a distributed ingestion lock.
	s.historyQ.On("Rollback").Return(nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: startState{}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *ResumeTestTestSuite) TestFastForwardCaptiveCore() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(101), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(101), nil)

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(102)).Return(true, nil).Once()
	s.ledgerBackend.On("GetLatestLedgerSequence").Return(uint32(99), nil).Once()
	// GetLedger will fast-forward to the latest sequence in a backend
	s.ledgerBackend.On("GetLedger", uint32(99)).Return(true, xdr.LedgerCloseMeta{}, nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 100}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 99},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

func (s *ResumeTestTestSuite) mockSuccessfulIngestion() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(101), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(101), nil)

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(102)).Return(true, nil).Once()
	s.ledgerBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	s.runner.On("RunAllProcessorsOnLedger", uint32(102)).Return(
		ingest.StatsChangeProcessorResults{},
		processorsRunDurations{},
		processors.StatsLedgerTransactionProcessorResults{},
		processorsRunDurations{},
		nil,
	).Once()
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

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(101)).Return(true, nil).Once()
	s.ledgerBackend.On("GetLatestLedgerSequence").Return(uint32(111), nil).Once()

	s.runner.On("RunAllProcessorsOnLedger", uint32(101)).Return(
		ingest.StatsChangeProcessorResults{},
		processorsRunDurations{},
		processors.StatsLedgerTransactionProcessorResults{},
		processorsRunDurations{},
		nil,
	).Once()
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

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(101)).Return(true, nil).Once()
	s.ledgerBackend.On("GetLatestLedgerSequence").Return(uint32(100), nil).Once()
	// Fast forward the backend
	s.ledgerBackend.On("GetLedger", uint32(100)).Return(true, xdr.LedgerCloseMeta{}, nil).Once()

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

func (s *ResumeTestTestSuite) TestFarBehind() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(200), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil)

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(201)).Return(true, nil).Once()
	s.ledgerBackend.On("GetLatestLedgerSequence").Return(uint32(102), nil).Once()
	// Fast forward the backend
	s.ledgerBackend.On("GetLedger", uint32(102)).Return(true, xdr.LedgerCloseMeta{}, nil).Once()

	next, err := resumeState{latestSuccessfullyProcessedLedger: 200}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			// Check the same ledger later
			node: resumeState{latestSuccessfullyProcessedLedger: 102},
			// Sleep because we learned the ledger is not there yet.
			sleepDuration: defaultSleep,
		},
		next,
	)
}
