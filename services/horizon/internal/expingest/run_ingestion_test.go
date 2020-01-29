package expingest

// import (
// 	"sort"
// 	"testing"
// 	"time"

// 	"github.com/jmoiron/sqlx"
// 	"github.com/stellar/go/exp/ingest/adapters"
// 	"github.com/stellar/go/exp/orderbook"
// 	"github.com/stellar/go/services/horizon/internal/db2/history"
// 	"github.com/stellar/go/support/db"
// 	"github.com/stellar/go/support/errors"
// 	"github.com/stellar/go/support/historyarchive"
// 	"github.com/stellar/go/xdr"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"github.com/stretchr/testify/suite"
// )

// type mockDBSession struct {
// 	mock.Mock
// }

// func (m *mockDBSession) Clone() *db.Session {
// 	args := m.Called()
// 	return args.Get(0).(*db.Session)
// }

// type mockDBQ struct {
// 	mock.Mock
// }

// func (m *mockDBQ) Begin() error {
// 	args := m.Called()
// 	return args.Error(0)
// }

// func (m *mockDBQ) Clone() *db.Session {
// 	args := m.Called()
// 	return args.Get(0).(*db.Session)
// }

// func (m *mockDBQ) Commit() error {
// 	args := m.Called()
// 	return args.Error(0)
// }

// func (m *mockDBQ) Rollback() error {
// 	args := m.Called()
// 	return args.Error(0)
// }

// func (m *mockDBQ) GetTx() *sqlx.Tx {
// 	args := m.Called()
// 	if args.Get(0) == nil {
// 		return nil
// 	}
// 	return args.Get(0).(*sqlx.Tx)
// }

// func (m *mockDBQ) GetLastLedgerExpIngest() (uint32, error) {
// 	args := m.Called()
// 	return args.Get(0).(uint32), args.Error(1)
// }

// func (m *mockDBQ) GetExpIngestVersion() (int, error) {
// 	args := m.Called()
// 	return args.Get(0).(int), args.Error(1)
// }

// func (m *mockDBQ) UpdateLastLedgerExpIngest(sequence uint32) error {
// 	args := m.Called(sequence)
// 	return args.Error(0)
// }

// func (m *mockDBQ) UpdateExpStateInvalid(invalid bool) error {
// 	args := m.Called(invalid)
// 	return args.Error(0)
// }

// func (m *mockDBQ) UpdateExpIngestVersion(version int) error {
// 	args := m.Called(version)
// 	return args.Error(0)
// }

// func (m *mockDBQ) GetExpStateInvalid() (bool, error) {
// 	args := m.Called()
// 	return args.Get(0).(bool), args.Error(1)
// }

// func (m *mockDBQ) GetAllOffers() ([]history.Offer, error) {
// 	args := m.Called()
// 	return args.Get(0).([]history.Offer), args.Error(1)
// }

// func (m *mockDBQ) GetLatestLedger() (uint32, error) {
// 	args := m.Called()
// 	return args.Get(0).(uint32), args.Error(1)
// }

// func (m *mockDBQ) TruncateExpingestStateTables() error {
// 	args := m.Called()
// 	return args.Error(0)
// }

// func (m *mockDBQ) DeleteRangeAll(start, end int64) error {
// 	args := m.Called(start, end)
// 	return args.Error(0)
// }

// type mockIngestSession struct {
// 	mock.Mock
// }

// func (m *mockIngestSession) Run() error {
// 	args := m.Called()
// 	return args.Error(0)
// }

// func (m *mockIngestSession) RunFromCheckpoint(checkpointLedger uint32) error {
// 	args := m.Called(checkpointLedger)
// 	return args.Error(0)
// }

// func (m *mockIngestSession) Resume(ledgerSequence uint32) error {
// 	args := m.Called(ledgerSequence)
// 	return args.Error(0)
// }

// func (m *mockIngestSession) GetArchive() historyarchive.ArchiveInterface {
// 	args := m.Called()
// 	return args.Get(0).(historyarchive.ArchiveInterface)
// }

// func (m *mockIngestSession) GetLatestSuccessfullyProcessedLedger() (ledgerSequence uint32, processed bool) {
// 	args := m.Called()
// 	return args.Get(0).(uint32), args.Bool(1)
// }

// func (m *mockIngestSession) Shutdown() {
// 	m.Called()
// }

// type RunIngestionTestSuite struct {
// 	suite.Suite
// 	graph          *orderbook.OrderBookGraph
// 	session        *mockDBSession
// 	historyQ       *mockDBQ
// 	historyAdapter *adapters.MockHistoryArchiveAdapter
// 	ingestSession  *mockIngestSession
// 	system         *System
// 	expectedOffers []xdr.OfferEntry
// }

// func (s *RunIngestionTestSuite) SetupTest() {
// 	s.graph = orderbook.NewOrderBookGraph()
// 	s.session = &mockDBSession{}
// 	s.historyQ = &mockDBQ{}
// 	s.ingestSession = &mockIngestSession{}
// 	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
// 	s.system = &System{
// 		state:          state{systemState: initState},
// 		historySession: s.session,
// 		historyAdapter: s.historyAdapter,
// 		// historyQ:       s.historyQ,
// 		graph: s.graph,
// 	}
// 	s.expectedOffers = []xdr.OfferEntry{}

// 	s.Assert().Equal(initState, s.system.state.systemState)

// 	s.historyQ.On("GetTx").Return(nil).Once()
// 	s.historyQ.On("Begin").Return(nil).Once()
// }

// func (s *RunIngestionTestSuite) TearDownTest() {
// 	t := s.T()
// 	s.session.AssertExpectations(t)
// 	s.ingestSession.AssertExpectations(t)
// 	s.historyQ.AssertExpectations(t)
// 	assertions := assert.New(t)

// 	offers := s.graph.Offers()
// 	sort.Slice(offers, func(i, j int) bool {
// 		return offers[i].OfferId < offers[j].OfferId
// 	})
// 	assertions.Equal(s.expectedOffers, offers)
// }

// func (s *RunIngestionTestSuite) TestBeginReturnsError() {
// 	*s.historyQ = mockDBQ{}
// 	s.historyQ.On("GetTx").Return(nil).Once()
// 	s.historyQ.On("Begin").Return(errors.New("begin error")).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error in Begin: begin error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestGetLastLedgerExpIngestReturnsError() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(
// 		uint32(0),
// 		errors.New("last ledger error"),
// 	).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error getting last ingested ledger: last ledger error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestGetExpIngestVersionReturnsError() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(0, errors.New("version error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error getting exp ingest version: version error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestGetLatestLedgerReturnsError() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(0, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(0), errors.New("latest ledger error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error getting last history ledger sequence: latest ledger error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestUpdateLastLedgerExpIngestReturnsError() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(0), nextState.checkpointLedger)

// 	s.system.state = nextState

// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(errors.New("update last ledger error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error updating last ingested ledger: update last ledger error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestUpdateExpStateInvalidReturnsError() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(0), nextState.checkpointLedger)

// 	s.system.state = nextState

// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
// 	s.historyQ.On("UpdateExpStateInvalid", false).Return(
// 		errors.New("update exp state invalid error"),
// 	).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error updating state invalid value: update exp state invalid error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestTruncateTablesReturnsError() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(0), nextState.checkpointLedger)

// 	s.system.state = nextState

// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
// 	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
// 	s.historyQ.On("TruncateExpingestStateTables").Return(errors.New("truncate error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error clearing ingest tables: truncate error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// // TestNoExpIngestUpgradeHistoryLedgerGreaterThanLatestCheckpoint is testing a scenario
// // when user upgrades to the new system but latest history ledger is
// // greater than the latest checkpoint ledger. In such case we just wait for the next
// // checkpoint.
// func (s *RunIngestionTestSuite) TestNoExpIngestUpgradeHistoryLedgerGreaterThanLatestCheckpoint() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(waitForCheckpointState, nextState.systemState)
// }

// // TestNoExpIngestUpgradeHistoryLedgerLessThanLatestCheckpoint is testing a scenario
// // when user upgrades to the new system but latest history ledger is less
// // than the latest checkpoint ledger. In such case we catchup history ledgers
// // to the checkpoint ledger.
// func (s *RunIngestionTestSuite) TestNoExpIngestUpgradeHistoryLedgerLessThanLatestCheckpoint() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(50), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(ingestHistoryRangeState, nextState.systemState)
// 	s.Assert().Equal(uint32(51), nextState.rangeFromLedger)
// 	s.Assert().Equal(uint32(63), nextState.rangeToLedger)
// }

// // TestNoExpIngestUpgradeHistoryLedgerLessThanLatestCheckpoint is testing a scenario
// // when user upgrades to the new system but latest history ledger is less
// // than the latest checkpoint ledger. In such case we buildStateAndResume
// // but from the latest checkpoint as of this state call. This is to prevent
// // situations when after state transition the new checkpoint is created. This
// // would skip 64 ledgers in history tables.
// func (s *RunIngestionTestSuite) TestNoExpIngestUpgradeHistoryLedgerEqualLatestCheckpoint() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(63), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(63), nextState.checkpointLedger)

// 	s.system.state = nextState

// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
// 	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
// 	s.historyQ.On("TruncateExpingestStateTables").Return(nil).Once()
// 	s.ingestSession.On("RunFromCheckpoint", uint32(63)).Return(nil).Once()

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(shutdownState, nextState.systemState)
// }

// // TestUpgradeHistoryLedgerGreaterThanLatestCheckpoint is testing a scenario
// // when CurrentVersion has been upgraded but latest history ledger is greater
// // than the latest checkpoint ledger. In such case we just wait for the next
// // checkpoint.
// func (s *RunIngestionTestSuite) TestUpgradeHistoryLedgerGreaterThanLatestCheckpoint() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(63), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(waitForCheckpointState, nextState.systemState)
// }

// // TestUpgradeHistoryLedgerLessThanLatestCheckpoint is testing a scenario
// // when CurrentVersion has been upgraded but latest history ledger is less
// // than the latest checkpoint ledger. In such case we catchup history ledgers
// // to the checkpoint ledger.
// func (s *RunIngestionTestSuite) TestUpgradeHistoryLedgerLessThanLatestCheckpoint() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(63), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(50), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(ingestHistoryRangeState, nextState.systemState)
// 	s.Assert().Equal(uint32(51), nextState.rangeFromLedger)
// 	s.Assert().Equal(uint32(63), nextState.rangeToLedger)
// }

// // TestUpgradeHistoryLedgerLessThanLatestCheckpoint is testing a scenario
// // when CurrentVersion has been upgraded but latest history ledger is less
// // than the latest checkpoint ledger. In such case we buildStateAndResumeIngestion
// // but from the latest checkpoint as of this state call. This is to prevent
// // situations when after state transition the new checkpoint is created. This
// // would skip 64 ledgers in history tables.
// func (s *RunIngestionTestSuite) TestUpgradeHistoryLedgerEqualLatestCheckpoint() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(63), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(63), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(63), nextState.checkpointLedger)
// }

// // TestNoUpgradeHistoryLedgerGreaterThanExpIngestLatest is testing a scenario when
// // CurrentVersion in DB matches CurrentVersion in the code so system starts resume
// // path but last history ledger is greater than the last expingest ledger.
// // In such case reset the exp ledger sequence.
// func (s *RunIngestionTestSuite) TestNoUpgradeHistoryLedgerGreaterThanExpIngestLatest() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(63), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(65), nil).Once()

// 	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
// 	s.historyQ.On("Commit").Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// // TestNoUpgradeHistoryLedgerLessThanExpIngestLatest is testing a scenario when
// // CurrentVersion in DB matches CurrentVersion in the code so system starts resume
// // path but last history ledger is less than the last expingest ledger.
// // In such case we catchup history.
// func (s *RunIngestionTestSuite) TestNoUpgradeHistoryLedgerLessThanExpIngestLatest() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(63), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(60), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(ingestHistoryRangeState, nextState.systemState)
// 	s.Assert().Equal(uint32(61), nextState.rangeFromLedger)
// 	s.Assert().Equal(uint32(63), nextState.rangeToLedger)
// }

// func (s *RunIngestionTestSuite) TestRunReturnsErrorAfterProcessingNoLedgers() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()
// 	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
// 	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
// 	s.historyQ.On("TruncateExpingestStateTables").Return(nil).Once()
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.ingestSession.On("Run").Return(errors.New("run error")).Once()
// 	s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").Return(uint32(0), false).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(0), nextState.checkpointLedger)

// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "run error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestRunReturnsError() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()
// 	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
// 	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
// 	s.historyQ.On("TruncateExpingestStateTables").Return(nil).Once()
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Twice()
// 	s.ingestSession.On("Run").Return(errors.New("run error")).Once()
// 	s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").Return(uint32(3), true).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()
// 	s.ingestSession.On("Resume", uint32(4)).Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(0), nextState.checkpointLedger)

// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "run error")
// 	s.Assert().Equal(resumeState, nextState.systemState)
// 	s.Assert().Equal(uint32(3), nextState.latestSuccessfullyProcessedLedger)
// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	// Resume returns nil which means shut down
// 	s.Assert().Equal(shutdownState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestOutdatedIngestVersionNoHistoryData() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(0), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(0), nextState.checkpointLedger)
// }

// func (s *RunIngestionTestSuite) TestOutdatedIngestVersionHistoryBehindCheckpointLedger() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(127), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(ingestHistoryRangeState, nextState.systemState)
// 	s.Assert().Equal(uint32(101), nextState.rangeFromLedger)
// 	s.Assert().Equal(uint32(127), nextState.rangeToLedger)
// }

// func (s *RunIngestionTestSuite) TestOutdatedIngestVersionHistoryAfterCheckpointLedger() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(100), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(63), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(waitForCheckpointState, nextState.systemState)
// 	s.Assert().Equal(uint32(0), nextState.rangeFromLedger)
// 	s.Assert().Equal(uint32(0), nextState.rangeToLedger)
// }

// func (s *RunIngestionTestSuite) TestOutdatedIngestVersionHistoryEqualCheckpointLedger() {
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(127), nil).Once()
// 	s.historyAdapter.On("GetLatestLedgerSequence").Return(uint32(127), nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(127), nextState.checkpointLedger)
// }

// func (s *RunIngestionTestSuite) TestGetAllOffersReturnsError() {
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetAllOffers").Return(
// 		[]history.Offer{
// 			history.Offer{
// 				OfferID:      eurOffer.OfferId,
// 				SellerID:     eurOffer.SellerId.Address(),
// 				SellingAsset: eurOffer.Selling,
// 				BuyingAsset:  eurOffer.Buying,
// 				Amount:       eurOffer.Amount,
// 				Pricen:       int32(eurOffer.Price.N),
// 				Priced:       int32(eurOffer.Price.D),
// 				Price:        float64(eurOffer.Price.N) / float64(eurOffer.Price.D),
// 				Flags:        uint32(eurOffer.Flags),
// 			},
// 		},
// 		errors.New("get all offers error"),
// 	).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(3), nextState.latestSuccessfullyProcessedLedger)
// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "GetAllOffers error: get all offers error")
// 	s.Assert().Equal(initState, nextState.systemState)
// }

// func (s *RunIngestionTestSuite) TestGetAllOffersWithoutError() {
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Twice()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
// 	s.historyQ.On("GetLatestLedger").Return(uint32(3), nil).Once()
// 	s.historyQ.On("GetAllOffers").Return(
// 		[]history.Offer{
// 			history.Offer{
// 				OfferID:      eurOffer.OfferId,
// 				SellerID:     eurOffer.SellerId.Address(),
// 				SellingAsset: eurOffer.Selling,
// 				BuyingAsset:  eurOffer.Buying,
// 				Amount:       eurOffer.Amount,
// 				Pricen:       int32(eurOffer.Price.N),
// 				Priced:       int32(eurOffer.Price.D),
// 				Price:        float64(eurOffer.Price.N) / float64(eurOffer.Price.D),
// 				Flags:        uint32(eurOffer.Flags),
// 			},
// 			history.Offer{
// 				OfferID:      twoEurOffer.OfferId,
// 				SellerID:     twoEurOffer.SellerId.Address(),
// 				SellingAsset: twoEurOffer.Selling,
// 				BuyingAsset:  twoEurOffer.Buying,
// 				Amount:       twoEurOffer.Amount,
// 				Pricen:       int32(twoEurOffer.Price.N),
// 				Priced:       int32(twoEurOffer.Price.D),
// 				Price:        float64(twoEurOffer.Price.N) / float64(twoEurOffer.Price.D),
// 				Flags:        uint32(twoEurOffer.Flags),
// 			},
// 		},
// 		nil,
// 	).Once()
// 	s.ingestSession.On("Resume", uint32(4)).Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(uint32(3), nextState.latestSuccessfullyProcessedLedger)
// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(resumeState, nextState.systemState)
// 	s.Assert().Equal(uint32(3), nextState.latestSuccessfullyProcessedLedger)
// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	// Resume returns nil which means shut down
// 	s.Assert().Equal(shutdownState, nextState.systemState)

// 	s.expectedOffers = []xdr.OfferEntry{eurOffer, twoEurOffer}
// }

// func TestRunIngestionTestSuite(t *testing.T) {
// 	suite.Run(t, new(RunIngestionTestSuite))
// }

// type ResumeIngestionTestSuite struct {
// 	suite.Suite
// 	graph         *orderbook.OrderBookGraph
// 	session       *mockDBSession
// 	historyQ      *mockDBQ
// 	ingestSession *mockIngestSession
// 	system        *System
// }

// func (s *ResumeIngestionTestSuite) SetupTest() {
// 	s.graph = orderbook.NewOrderBookGraph()
// 	s.session = &mockDBSession{}
// 	s.historyQ = &mockDBQ{}
// 	s.ingestSession = &mockIngestSession{}
// 	s.system = &System{
// 		state:          state{systemState: resumeState, latestSuccessfullyProcessedLedger: 1},
// 		historySession: s.session,
// 		// historyQ:       s.historyQ,
// 		graph: s.graph,
// 	}

// 	s.Assert().Equal(resumeState, s.system.state.systemState)
// 	s.Assert().Equal(uint32(1), s.system.state.latestSuccessfullyProcessedLedger)

// 	s.historyQ.On("GetTx").Return(nil).Once()
// 	s.historyQ.On("Begin").Return(nil).Once()
// }

// func (s *ResumeIngestionTestSuite) TearDownTest() {
// 	t := s.T()
// 	s.session.AssertExpectations(t)
// 	s.ingestSession.AssertExpectations(t)
// 	s.historyQ.AssertExpectations(t)
// }

// func (s *ResumeIngestionTestSuite) TestResumeSucceeds() {
// 	s.ingestSession.On("Resume", uint32(2)).Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	// Resume returns nil which means shut down
// 	s.Assert().Equal(shutdownState, nextState.systemState)
// }

// func (s *ResumeIngestionTestSuite) TestResumeMakesProgress() {
// 	s.ingestSession.On("Resume", uint32(2)).Return(errors.New("first error")).Once()
// 	s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").
// 		Return(uint32(4), true).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()
// 	s.historyQ.On("GetTx").Return(nil).Once()
// 	s.historyQ.On("Begin").Return(nil).Once()
// 	s.ingestSession.On("Resume", uint32(5)).Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error returned from ingest.LiveSession: first error")
// 	s.Assert().Equal(resumeState, nextState.systemState)
// 	s.Assert().Equal(uint32(4), nextState.latestSuccessfullyProcessedLedger)
// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(shutdownState, nextState.systemState)
// }

// func (s *ResumeIngestionTestSuite) TestResumeDoesNotMakeProgress() {
// 	s.ingestSession.On("Resume", uint32(2)).Return(errors.New("first error")).Once()
// 	s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").
// 		Return(uint32(0), false).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()
// 	s.historyQ.On("GetTx").Return(nil).Once()
// 	s.historyQ.On("Begin").Return(nil).Once()
// 	s.ingestSession.On("Resume", uint32(2)).Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error returned from ingest.LiveSession: first error")
// 	s.Assert().Equal(resumeState, nextState.systemState)
// 	s.Assert().Equal(uint32(1), nextState.latestSuccessfullyProcessedLedger)
// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(shutdownState, nextState.systemState)
// }

// func (s *ResumeIngestionTestSuite) TestLedgerUpdatesOnlyIfProcessed() {
// 	s.ingestSession.On("Resume", uint32(2)).Return(errors.New("first error")).Once()
// 	s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").
// 		Return(uint32(5), false).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()
// 	s.historyQ.On("GetTx").Return(nil).Once()
// 	s.historyQ.On("Begin").Return(nil).Once()
// 	s.ingestSession.On("Resume", uint32(2)).Return(nil).Once()

// 	nextState, err := s.system.runCurrentState()
// 	s.Assert().Error(err)
// 	s.Assert().EqualError(err, "Error returned from ingest.LiveSession: first error")
// 	s.Assert().Equal(resumeState, nextState.systemState)
// 	s.Assert().Equal(uint32(1), nextState.latestSuccessfullyProcessedLedger)
// 	s.system.state = nextState

// 	nextState, err = s.system.runCurrentState()
// 	s.Assert().NoError(err)
// 	s.Assert().Equal(shutdownState, nextState.systemState)
// }

// func TestResumeIngestionTestSuite(t *testing.T) {
// 	suite.Run(t, new(ResumeIngestionTestSuite))
// }

// type SystemShutdownTestSuite struct {
// 	suite.Suite
// 	graph         *orderbook.OrderBookGraph
// 	session       *mockDBSession
// 	historyQ      *mockDBQ
// 	ingestSession *mockIngestSession
// 	system        *System
// }

// func (s *SystemShutdownTestSuite) SetupTest() {
// 	s.graph = orderbook.NewOrderBookGraph()
// 	s.session = &mockDBSession{}
// 	s.historyQ = &mockDBQ{}
// 	s.ingestSession = &mockIngestSession{}
// 	s.system = &System{
// 		historySession: s.session,
// 		// historyQ:       s.historyQ,
// 		graph:    s.graph,
// 		shutdown: make(chan struct{}),
// 	}
// }

// func (s *SystemShutdownTestSuite) TearDownTest() {
// 	t := s.T()
// 	s.session.AssertExpectations(t)
// 	s.ingestSession.AssertExpectations(t)
// 	s.historyQ.AssertExpectations(t)
// }

// func (s *SystemShutdownTestSuite) TestShutdownSucceeds() {
// 	s.ingestSession.On("Shutdown").Return(nil).Once()
// 	done := make(chan struct{})
// 	go func() {
// 		defer close(done)
// 		select {
// 		case <-s.system.shutdown:
// 			s.Assert().True(true, "channel was closed")
// 		case <-time.After(2 * time.Second):
// 			s.Assert().Fail("channel should have been closed")
// 		}
// 	}()
// 	time.Sleep(100 * time.Millisecond)
// 	s.system.Shutdown()
// 	<-done
// }

// func TestSystemShutdownTestSuite(t *testing.T) {
// 	suite.Run(t, new(SystemShutdownTestSuite))
// }
