package expingest

import (
	"sort"
	"testing"

	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockDBSession struct {
	mock.Mock
}

func (m *mockDBSession) TruncateTables(tables []string) error {
	args := m.Called(tables)
	return args.Error(0)
}

func (m *mockDBSession) Clone() *db.Session {
	args := m.Called()
	return args.Get(0).(*db.Session)
}

type mockDBQ struct {
	mock.Mock
}

func (m *mockDBQ) Begin() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockDBQ) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockDBQ) GetLastLedgerExpIngest() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *mockDBQ) GetExpIngestVersion() (int, error) {
	args := m.Called()
	return args.Get(0).(int), args.Error(1)
}

func (m *mockDBQ) UpdateLastLedgerExpIngest(sequence uint32) error {
	args := m.Called(sequence)
	return args.Error(0)
}

func (m *mockDBQ) UpdateExpStateInvalid(invalid bool) error {
	args := m.Called(invalid)
	return args.Error(0)
}

func (m *mockDBQ) GetExpStateInvalid() (bool, error) {
	args := m.Called()
	return args.Get(0).(bool), args.Error(1)
}

func (m *mockDBQ) GetAllOffers() ([]history.Offer, error) {
	args := m.Called()
	return args.Get(0).([]history.Offer), args.Error(1)
}

type mockIngestSession struct {
	mock.Mock
}

func (m *mockIngestSession) Run() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockIngestSession) Resume(ledgerSequence uint32) error {
	args := m.Called(ledgerSequence)
	return args.Error(0)
}

func (m *mockIngestSession) GetArchive() historyarchive.ArchiveInterface {
	args := m.Called()
	return args.Get(0).(historyarchive.ArchiveInterface)
}

func (m *mockIngestSession) GetLatestSuccessfullyProcessedLedger() (ledgerSequence uint32, processed bool) {
	args := m.Called()
	return args.Get(0).(uint32), args.Bool(1)
}

func (m *mockIngestSession) Shutdown() {
	m.Called()
}

type retryFunc func(func() error)

func (f retryFunc) onError(lambda func() error) {
	f(lambda)
}

func expectError(assertions *assert.Assertions, expectedError string) retryFunc {
	return func(f func() error) {
		err := f()

		if expectedError == "" {
			assertions.NoError(err)
		} else {
			assertions.EqualError(errors.Cause(err), expectedError)
		}
	}
}

type RunIngestionTestSuite struct {
	suite.Suite
	graph          *orderbook.OrderBookGraph
	session        *mockDBSession
	historyQ       *mockDBQ
	ingestSession  *mockIngestSession
	system         *System
	expectedOffers []xdr.OfferEntry
}

func (s *RunIngestionTestSuite) SetupTest() {
	s.graph = orderbook.NewOrderBookGraph()
	s.session = &mockDBSession{}
	s.historyQ = &mockDBQ{}
	s.ingestSession = &mockIngestSession{}
	s.system = &System{
		session:        s.ingestSession,
		historySession: s.session,
		historyQ:       s.historyQ,
		graph:          s.graph,
	}
	s.expectedOffers = []xdr.OfferEntry{}
}

func (s *RunIngestionTestSuite) TearDownTest() {
	s.system.Run()

	t := s.T()
	s.session.AssertExpectations(t)
	s.ingestSession.AssertExpectations(t)
	s.historyQ.AssertExpectations(t)
	assertions := assert.New(t)

	offers := s.graph.Offers()
	sort.Slice(offers, func(i, j int) bool {
		return offers[i].OfferId < offers[j].OfferId
	})
	assertions.Equal(s.expectedOffers, offers)
}

func (s *RunIngestionTestSuite) TestBeginReturnsError() {
	s.historyQ.On("Begin").Return(errors.New("begin error")).Once()
	s.system.retry = expectError(s.Assert(), "begin error")
}

func (s *RunIngestionTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(
		uint32(0),
		errors.New("last ledger error"),
	).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "last ledger error")
}

func (s *RunIngestionTestSuite) TestGetExpIngestVersionReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(0, errors.New("version error")).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "version error")
}

func (s *RunIngestionTestSuite) TestUpdateLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(
		errors.New("update last ledger error"),
	).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "update last ledger error")
}

func (s *RunIngestionTestSuite) TestUpdateExpStateInvalidReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(
		errors.New("update exp state invalid error"),
	).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "update exp state invalid error")
}

func (s *RunIngestionTestSuite) TestTruncateTablesReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.session.On("TruncateTables", history.ExperimentalIngestionTables).Return(
		errors.New("truncate error"),
	).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "truncate error")
}

func (s *RunIngestionTestSuite) TestRunReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.session.On("TruncateTables", history.ExperimentalIngestionTables).Return(nil).Once()
	s.ingestSession.On("Run").Return(errors.New("run error")).Once()
	s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").Return(uint32(3), true).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.ingestSession.On("Resume", uint32(4)).Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "")
}

func (s *RunIngestionTestSuite) TestOutdatedIngestVersion() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion-1, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(0)).Return(nil).Once()
	s.historyQ.On("UpdateExpStateInvalid", false).Return(nil).Once()
	s.session.On("TruncateTables", history.ExperimentalIngestionTables).Return(nil).Once()
	s.ingestSession.On("Run").Return(nil).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.ingestSession.On("Resume", uint32(4)).Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "")
}

func (s *RunIngestionTestSuite) TestGetAllOffersReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetAllOffers").Return(
		[]history.Offer{
			history.Offer{
				OfferID:      eurOffer.OfferId,
				SellerID:     eurOffer.SellerId.Address(),
				SellingAsset: eurOffer.Selling,
				BuyingAsset:  eurOffer.Buying,
				Amount:       eurOffer.Amount,
				Pricen:       int32(eurOffer.Price.N),
				Priced:       int32(eurOffer.Price.D),
				Price:        float64(eurOffer.Price.N) / float64(eurOffer.Price.D),
				Flags:        uint32(eurOffer.Flags),
			},
		},
		errors.New("get all offers error"),
	).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "get all offers error")
}

func (s *RunIngestionTestSuite) TestGetAllOffersWithoutError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(3), nil).Once()
	s.historyQ.On("GetExpIngestVersion").Return(CurrentVersion, nil).Once()
	s.historyQ.On("GetAllOffers").Return(
		[]history.Offer{
			history.Offer{
				OfferID:      eurOffer.OfferId,
				SellerID:     eurOffer.SellerId.Address(),
				SellingAsset: eurOffer.Selling,
				BuyingAsset:  eurOffer.Buying,
				Amount:       eurOffer.Amount,
				Pricen:       int32(eurOffer.Price.N),
				Priced:       int32(eurOffer.Price.D),
				Price:        float64(eurOffer.Price.N) / float64(eurOffer.Price.D),
				Flags:        uint32(eurOffer.Flags),
			},
			history.Offer{
				OfferID:      twoEurOffer.OfferId,
				SellerID:     twoEurOffer.SellerId.Address(),
				SellingAsset: twoEurOffer.Selling,
				BuyingAsset:  twoEurOffer.Buying,
				Amount:       twoEurOffer.Amount,
				Pricen:       int32(twoEurOffer.Price.N),
				Priced:       int32(twoEurOffer.Price.D),
				Price:        float64(twoEurOffer.Price.N) / float64(twoEurOffer.Price.D),
				Flags:        uint32(twoEurOffer.Flags),
			},
		},
		nil,
	).Once()
	s.historyQ.On("Rollback").Return(nil).Once()
	s.ingestSession.On("Resume", uint32(4)).Return(nil).Once()
	s.system.retry = expectError(s.Assert(), "")
	s.expectedOffers = []xdr.OfferEntry{eurOffer, twoEurOffer}
}

func TestRunIngestionTestSuite(t *testing.T) {
	suite.Run(t, new(RunIngestionTestSuite))
}

type ResumeIngestionTestSuite struct {
	suite.Suite
	graph            *orderbook.OrderBookGraph
	session          *mockDBSession
	historyQ         *mockDBQ
	ingestSession    *mockIngestSession
	system           *System
	attempts         int
	expectedAttempts int
}

func (s *ResumeIngestionTestSuite) SetupTest() {
	s.graph = orderbook.NewOrderBookGraph()
	s.session = &mockDBSession{}
	s.historyQ = &mockDBQ{}
	s.ingestSession = &mockIngestSession{}
	s.attempts = 0
	s.expectedAttempts = 0
	s.system = &System{
		session:        s.ingestSession,
		historySession: s.session,
		historyQ:       s.historyQ,
		graph:          s.graph,
	}
}

func (s *ResumeIngestionTestSuite) TearDownTest() {
	s.system.resumeFromLedger(1)

	t := s.T()
	s.session.AssertExpectations(t)
	s.ingestSession.AssertExpectations(t)
	s.historyQ.AssertExpectations(t)
	if s.attempts != s.expectedAttempts {
		t.Fatalf("expected only %v attempts but got %v", s.expectedAttempts, s.attempts)
	}
}

func (s *ResumeIngestionTestSuite) TestResumeSucceeds() {
	s.ingestSession.On("Resume", uint32(2)).Return(nil).Once()
	s.system.retry = retryFunc(func(f func() error) {
		s.Assert().NoError(f())
	})
}

func (s *ResumeIngestionTestSuite) TestResumeMakesProgress() {
	s.system.retry = retryFunc(func(f func() error) {
		for {
			s.attempts++
			var expectedError string

			if s.attempts == 1 {
				expectedError = "first error"
				s.ingestSession.On("Resume", uint32(2)).Return(errors.New(expectedError)).Once()
				s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").
					Return(uint32(4), true).Once()
			} else if s.attempts == 2 {
				s.ingestSession.On("Resume", uint32(5)).Return(nil).Once()
			}

			err := f()
			s.ingestSession.AssertExpectations(s.T())

			if expectedError == "" {
				s.Assert().NoError(err)
				break
			} else {
				s.Assert().EqualError(errors.Cause(err), expectedError)
			}
		}
	})
	s.expectedAttempts = 2
}

func (s *ResumeIngestionTestSuite) TestResumeDoesNotMakeProgress() {
	s.system.retry = retryFunc(func(f func() error) {
		for {
			s.attempts++
			var expectedError string

			if s.attempts == 1 {
				expectedError = "first error"
				s.ingestSession.On("Resume", uint32(2)).Return(errors.New(expectedError)).Once()
				s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").
					Return(uint32(0), false).Once()
			} else if s.attempts == 2 {
				s.ingestSession.On("Resume", uint32(2)).Return(nil).Once()
			}

			err := f()
			s.ingestSession.AssertExpectations(s.T())

			if expectedError == "" {
				s.Assert().NoError(err)
				break
			} else {
				s.Assert().EqualError(errors.Cause(err), expectedError)
			}
		}
	})
	s.expectedAttempts = 2
}

func (s *ResumeIngestionTestSuite) TestLedgerUpdatesOnlyIfProcessed() {
	s.system.retry = retryFunc(func(f func() error) {
		for {
			s.attempts++
			var expectedError string

			if s.attempts == 1 {
				expectedError = "first error"
				s.ingestSession.On("Resume", uint32(2)).Return(errors.New(expectedError)).Once()
				s.ingestSession.On("GetLatestSuccessfullyProcessedLedger").Return(uint32(5), false)
			} else if s.attempts == 2 {
				s.ingestSession.On("Resume", uint32(2)).Return(nil).Once()
			}

			err := f()
			s.ingestSession.AssertExpectations(s.T())

			if expectedError == "" {
				s.Assert().NoError(err)
				break
			} else {
				s.Assert().EqualError(errors.Cause(err), expectedError)
			}
		}
	})
	s.expectedAttempts = 2
}

func TestResumeIngestionTestSuite(t *testing.T) {
	suite.Run(t, new(ResumeIngestionTestSuite))
}
