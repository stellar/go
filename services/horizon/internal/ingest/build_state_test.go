//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package ingest

import (
	"context"
	"testing"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestBuildStateTestSuite(t *testing.T) {
	suite.Run(t, new(BuildStateTestSuite))
}

type BuildStateTestSuite struct {
	suite.Suite
	historyQ          *mockDBQ
	historyAdapter    *mockHistoryArchiveAdapter
	ledgerBackend     *ledgerbackend.MockDatabaseBackend
	system            *system
	runner            *mockProcessorsRunner
	stellarCoreClient *mockStellarCoreClient
	checkpointLedger  uint32
	lastLedger        uint32
}

func (s *BuildStateTestSuite) SetupTest() {
	s.historyQ = &mockDBQ{}
	s.runner = &mockProcessorsRunner{}
	s.historyAdapter = &mockHistoryArchiveAdapter{}
	s.ledgerBackend = &ledgerbackend.MockDatabaseBackend{}
	s.stellarCoreClient = &mockStellarCoreClient{}
	s.checkpointLedger = uint32(63)
	s.lastLedger = 0
	s.system = &system{
		ctx:               context.Background(),
		historyQ:          s.historyQ,
		historyAdapter:    s.historyAdapter,
		ledgerBackend:     s.ledgerBackend,
		runner:            s.runner,
		stellarCoreClient: s.stellarCoreClient,
	}
	s.system.initMetrics()

	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *BuildStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.runner.AssertExpectations(t)
	s.stellarCoreClient.AssertExpectations(t)
	s.ledgerBackend.AssertExpectations(t)
}

func (s *BuildStateTestSuite) mockCommonHistoryQ() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateExpingestStateTables").Return(nil).Once()
	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(62),
	).Return(nil).Once()
	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(63)).Return(true, nil).Once()
}

func (s *BuildStateTestSuite) TestCheckPointLedgerIsZero() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}

	next, err := buildState{checkpointLedger: 0}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "unexpected checkpointLedger value")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, errors.New("my error")).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}
func (s *BuildStateTestSuite) TestGetExpIngestVersionReturnsError() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, errors.New("my error")).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting exp ingest version: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestAnotherInstanceHasCompletedBuildState() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.checkpointLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateLastLedgerExpIngestReturnsError() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.lastLedger).Return(errors.New("my error")).Once()
	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(62),
	).Return(nil).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error updating last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateExpStateInvalidReturnsError() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(errors.New("my error")).Once()
	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(62),
	).Return(nil).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error updating state invalid value: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestTruncateExpingestStateTablesReturnsError() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateExpingestStateTables").Return(errors.New("my error")).Once()

	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(62),
	).Return(nil).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error clearing ingest tables: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestRangeNotPreparedFailPrepare() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateExpingestStateTables").Return(nil).Once()

	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(62),
	).Return(nil).Once()

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(63)).Return(false, nil).Once()
	s.ledgerBackend.On("PrepareRange", ledgerbackend.UnboundedRange(63)).Return(errors.New("my error")).Once()
	// Rollback twice (first one mocked in SetupTest) because we want to release
	// a distributed ingestion lock.
	s.historyQ.On("Rollback").Return(nil).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "error preparing range: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestRangeNotPreparedSuccessPrepare() {
	s.historyQ.On("GetLastLedgerExpIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateExpingestStateTables").Return(nil).Once()

	s.stellarCoreClient.On(
		"SetCursor",
		mock.AnythingOfType("*context.timerCtx"),
		defaultCoreCursorName,
		int32(62),
	).Return(nil).Once()

	s.ledgerBackend.On("IsPrepared", ledgerbackend.UnboundedRange(63)).Return(false, nil).Once()
	s.ledgerBackend.On("PrepareRange", ledgerbackend.UnboundedRange(63)).Return(nil).Once()
	// Rollback twice (first one mocked in SetupTest) because we want to release
	// a distributed ingestion lock.
	s.historyQ.On("Rollback").Return(nil).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{
		// suggestedCheckpoint is set. See startSuggestedCheckpoint for more info.
		suggestedCheckpoint: 63,
	}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestRunHistoryArchiveIngestionReturnsError() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(ingest.StatsChangeProcessorResults{}, errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error ingesting history archive: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateLastLedgerExpIngestAfterIngestReturnsError() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(ingest.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateExpIngestVersion", CurrentVersion).
		Return(nil).
		Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.checkpointLedger).
		Return(errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error updating last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateExpIngestVersionIngestReturnsError() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(ingest.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateExpIngestVersion", CurrentVersion).
		Return(errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error updating ingest version: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateCommitReturnsError() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(ingest.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateExpIngestVersion", CurrentVersion).
		Return(nil).
		Once()
	s.historyQ.On("Commit").
		Return(errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error committing db transaction: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestBuildStateSucceeds() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(ingest.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateExpIngestVersion", CurrentVersion).
		Return(nil).
		Once()
	s.historyQ.On("Commit").
		Return(nil).
		Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: s.checkpointLedger},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

func (s *BuildStateTestSuite) TestUpdateCommitReturnsErrorStop() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(ingest.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateExpIngestVersion", CurrentVersion).
		Return(nil).
		Once()
	s.historyQ.On("Commit").
		Return(errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger, stop: true}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error committing db transaction: my error")
	s.Assert().Equal(transition{node: stopState{}, sleepDuration: 0}, next)
}

func (s *BuildStateTestSuite) TestBuildStateSucceedStop() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(ingest.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateExpIngestVersion", CurrentVersion).
		Return(nil).
		Once()
	s.historyQ.On("Commit").
		Return(nil).
		Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger, stop: true}.run(s.system)

	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          stopState{},
			sleepDuration: 0,
		},
		next,
	)
}
