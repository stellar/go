//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite
package expingest

import (
	"context"
	"testing"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/suite"
)

func TestInitStateTestSuite(t *testing.T) {
	suite.Run(t, new(InitStateTestSuite))
}

type InitStateTestSuite struct {
	suite.Suite
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	system         *system
}

func (s *InitStateTestSuite) SetupTest() {
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.system = &system{
		ctx:            context.Background(),
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
	}
	s.system.initMetrics()

	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *InitStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
}

func (s *InitStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestGetExpIngestVersionReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting exp ingest version: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestCurrentVersionIsOutdated() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(1), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion+1, nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: stopState{}, sleepDuration: 0}, next)
}

func (s *InitStateTestSuite) TestGetLatestLedgerReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), errors.New("my error")).Once()

	next, err := startState{}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last history ledger sequence: my error")
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

func (s *InitStateTestSuite) TestBuildStateEmptyDatabase() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: buildState{checkpointLedger: 63}, sleepDuration: defaultSleep},
		next,
	)
}

func (s *InitStateTestSuite) TestBuildStateEmptyDatabaseFromSuggestedCheckpoint() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

	next, err := startState{suggestedCheckpoint: 127}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: buildState{checkpointLedger: 127}, sleepDuration: defaultSleep},
		next,
	)
}

// TestBuildStateWait is testing the case when:
// * the ingest system version has been incremented or no expingest ledger,
// * the old system is in front of the latest checkpoint.
func (s *InitStateTestSuite) TestBuildStateWait() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: waitForCheckpointState{}, sleepDuration: 0},
		next,
	)
}

// TestBuildStateCatchup is testing the case when:
// * the ingest system version has been incremented or no expingest ledger,
// * the old system is behind the latest checkpoint.
func (s *InitStateTestSuite) TestBuildStateCatchup() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(127), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          historyRangeState{fromLedger: 101, toLedger: 127},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestBuildStateOldHistory is testing the case when:
// * the ingest system version has been incremented or no expingest ledger,
// * the old system latest ledger is equal to the latest checkpoint.
func (s *InitStateTestSuite) TestBuildStateOldHistory() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(127), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(127), nil).Once()

	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(127), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          buildState{checkpointLedger: 127},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestResumeStateInFront is testing the case when:
// * state doesn't need to be rebuilt,
// * history is in front of expingest.
func (s *InitStateTestSuite) TestResumeStateInFront() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(130), nil).Once()

	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(transition{node: startState{}, sleepDuration: defaultSleep}, next)
}

// TestResumeStateBehind is testing the case when:
// * state doesn't need to be rebuilt,
// * history is behind of expingest.
func (s *InitStateTestSuite) TestResumeStateBehind() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(130), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          historyRangeState{fromLedger: 101, toLedger: 130},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestResumeStateBehindHistory0 is testing the case when:
// * state doesn't need to be rebuilt or was just rebuilt,
// * there are no ledgers in history tables.
// In such case we load offers and continue ingesting the next ledger.
func (s *InitStateTestSuite) TestResumeStateBehindHistory0() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(130), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 130},
			sleepDuration: defaultSleep,
		},
		next,
	)
}

// TestResumeStateBehind is testing the case when:
// * state doesn't need to be rebuilt,
// * history is in sync with expingest.
func (s *InitStateTestSuite) TestResumeStateSync() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(130), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetLatestLedger").Return(uint32(130), nil).Once()

	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{
			node:          resumeState{latestSuccessfullyProcessedLedger: 130},
			sleepDuration: defaultSleep,
		},
		next,
	)
}
