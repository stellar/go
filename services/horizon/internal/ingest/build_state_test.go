//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package ingest

import (
	"context"
	"testing"

	"github.com/stellar/go/ingest/adapters"
	"github.com/stellar/go/ingest/io"
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
	historyAdapter    *adapters.MockHistoryArchiveAdapter
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
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
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
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateIngestStateTables").Return(nil).Once()
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

func (s *BuildStateTestSuite) TestGetLastLedgerIngestReturnsError() {
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, errors.New("my error")).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}
func (s *BuildStateTestSuite) TestGetIngestVersionReturnsError() {
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, errors.New("my error")).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting ingestion version: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestAnotherInstanceHasCompletedBuildState() {
	s.historyQ.On("GetLastLedgerIngest").Return(s.checkpointLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, nil).Once()

	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateLastLedgerIngestReturnsError() {
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.lastLedger).Return(errors.New("my error")).Once()
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
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.lastLedger).Return(nil).Once()
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

func (s *BuildStateTestSuite) TestTruncateIngestStateTablesReturnsError() {
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateIngestStateTables").Return(errors.New("my error")).Once()

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
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateIngestStateTables").Return(nil).Once()

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
	s.historyQ.On("GetLastLedgerIngest").Return(s.lastLedger, nil).Once()
	s.historyQ.On("GetIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.lastLedger).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.historyQ.On("TruncateIngestStateTables").Return(nil).Once()

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
		Return(io.StatsChangeProcessorResults{}, errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error ingesting history archive: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateLastLedgerIngestAfterIngestReturnsError() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(io.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateIngestVersion", CurrentVersion).
		Return(nil).
		Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.checkpointLedger).
		Return(errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error updating last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateIngestVersionIngestReturnsError() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(io.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateIngestVersion", CurrentVersion).
		Return(errors.New("my error")).
		Once()
	next, err := buildState{checkpointLedger: s.checkpointLedger}.run(s.system)

	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error updating ingestion version: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *BuildStateTestSuite) TestUpdateCommitReturnsError() {
	s.mockCommonHistoryQ()
	s.runner.
		On("RunHistoryArchiveIngestion", s.checkpointLedger).
		Return(io.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateIngestVersion", CurrentVersion).
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
		Return(io.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateIngestVersion", CurrentVersion).
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
		Return(io.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateIngestVersion", CurrentVersion).
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
		Return(io.StatsChangeProcessorResults{}, nil).
		Once()
	s.historyQ.On("UpdateLastLedgerIngest", s.checkpointLedger).
		Return(nil).
		Once()
	s.historyQ.On("UpdateIngestVersion", CurrentVersion).
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
