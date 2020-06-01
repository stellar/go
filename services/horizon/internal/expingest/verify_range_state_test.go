package expingest

import (
	"database/sql"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/exp/ingest/adapters"
	ingestio "github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func TestVerifyRangeStateTestSuite(t *testing.T) {
	suite.Run(t, new(VerifyRangeStateTestSuite))
}

type VerifyRangeStateTestSuite struct {
	suite.Suite
	historyQ       *mockDBQ
	historyAdapter *adapters.MockHistoryArchiveAdapter
	runner         *mockProcessorsRunner
	system         *System
}

func (s *VerifyRangeStateTestSuite) SetupTest() {
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &adapters.MockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.system = &System{
		historyQ:       s.historyQ,
		historyAdapter: s.historyAdapter,
		runner:         s.runner,
	}
	s.system.initMetrics()

	s.historyQ.On("Rollback").Return(nil).Once()
}

func (s *VerifyRangeStateTestSuite) TearDownTest() {
	t := s.T()
	s.historyQ.AssertExpectations(t)
	s.historyAdapter.AssertExpectations(t)
	s.runner.AssertExpectations(t)
}

func (s *VerifyRangeStateTestSuite) TestInvalidRange() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}

	next, err := verifyRangeState{fromLedger: 0, toLedger: 0}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 0]")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)

	next, err = verifyRangeState{fromLedger: 0, toLedger: 100}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [0, 100]")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)

	next, err = verifyRangeState{fromLedger: 100, toLedger: 0}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 0]")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)

	next, err = verifyRangeState{fromLedger: 100, toLedger: 99}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "invalid range: [100, 99]")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
}

func (s *VerifyRangeStateTestSuite) TestBeginReturnsError() {
	// Recreate mock in this single test to remove Rollback assertion.
	*s.historyQ = mockDBQ{}
	s.historyQ.On("Begin").Return(errors.New("my error")).Once()

	next, err := verifyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error starting a transaction: my error")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
}

func (s *VerifyRangeStateTestSuite) TestGetLastLedgerExpIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("my error")).Once()

	next, err := verifyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
}

func (s *VerifyRangeStateTestSuite) TestGetLastLedgerExpIngestNonEmpty() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(100), nil).Once()

	next, err := verifyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Database not empty")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
}

func (s *VerifyRangeStateTestSuite) TestRunHistoryArchiveIngestionReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()

	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(ingestio.StatsChangeProcessorResults{}, errors.New("my error")).Once()

	next, err := verifyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error ingesting history archive: my error")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
}

func (s *VerifyRangeStateTestSuite) TestSuccess() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(ingestio.StatsChangeProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(100)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	for i := uint32(101); i <= 200; i++ {
		s.historyQ.On("Begin").Return(nil).Once()
		s.runner.On("RunAllProcessorsOnLedger", i).Return(ingestio.StatsChangeProcessorResults{},
			ingestio.StatsLedgerTransactionProcessorResults{}, nil).Once()
		s.historyQ.On("UpdateLastLedgerExpIngest", i).Return(nil).Once()
		s.historyQ.On("Commit").Return(nil).Once()
	}

	next, err := verifyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
}

func (s *VerifyRangeStateTestSuite) TestSuccessWithVerify() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()
	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(ingestio.StatsChangeProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerExpIngest", uint32(100)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	for i := uint32(101); i <= 110; i++ {
		s.historyQ.On("Begin").Return(nil).Once()
		s.runner.On("RunAllProcessorsOnLedger", i).Return(ingestio.StatsChangeProcessorResults{},
			ingestio.StatsLedgerTransactionProcessorResults{}, nil).Once()
		s.historyQ.On("UpdateLastLedgerExpIngest", i).Return(nil).Once()
		s.historyQ.On("Commit").Return(nil).Once()
	}

	clonedQ := &mockDBQ{}
	s.historyQ.On("CloneIngestionQ").Return(clonedQ).Once()

	clonedQ.On("BeginTx", mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(0).(*sql.TxOptions)
		s.Assert().Equal(sql.LevelRepeatableRead, arg.Isolation)
		s.Assert().True(arg.ReadOnly)
	}).Return(nil).Once()
	clonedQ.On("Rollback").Return(nil).Once()
	clonedQ.On("GetLastLedgerExpIngestNonBlocking").Return(uint32(63), nil).Once()
	mockChangeReader := &ingestio.MockChangeReader{}
	mockChangeReader.On("Close").Return(nil).Once()
	mockAccountID := "GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX"
	accountChange := ingestio.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:  xdr.MustAddress(mockAccountID),
					Balance:    xdr.Int64(600),
					Thresholds: [4]byte{1, 0, 0, 0},
				},
			},
			LastModifiedLedgerSeq: xdr.Uint32(62),
		},
	}
	offerChange := ingestio.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:  xdr.LedgerEntryTypeOffer,
				Offer: &eurOffer,
			},
			LastModifiedLedgerSeq: xdr.Uint32(62),
		},
	}
	mockChangeReader.On("Read").Return(accountChange, nil).Once()
	mockChangeReader.On("Read").Return(offerChange, nil).Once()
	mockChangeReader.On("Read").Return(ingestio.Change{}, io.EOF).Once()
	mockChangeReader.On("Read").Return(ingestio.Change{}, io.EOF).Once()
	s.historyAdapter.On("GetState", nil, uint32(63), 0).Return(mockChangeReader, nil).Once()
	mockAccount := history.AccountEntry{
		AccountID:          mockAccountID,
		Balance:            600,
		LastModifiedLedger: 62,
		MasterWeight:       1,
	}
	clonedQ.MockQAccounts.On("GetAccountsByIDs", []string{mockAccountID}).Return([]history.AccountEntry{mockAccount}, nil).Once()
	mockSigner := history.AccountSigner{
		Account: mockAccountID,
		Signer:  mockAccountID,
		Weight:  1,
	}
	clonedQ.MockQSigners.On("SignersForAccounts", []string{mockAccountID}).Return([]history.AccountSigner{mockSigner}, nil).Once()
	clonedQ.MockQSigners.On("CountAccounts").Return(1, nil).Once()
	mockOffer := history.Offer{
		SellerID:           eurOffer.SellerId.Address(),
		OfferID:            eurOffer.OfferId,
		SellingAsset:       eurOffer.Selling,
		BuyingAsset:        eurOffer.Buying,
		Amount:             eurOffer.Amount,
		Pricen:             int32(eurOffer.Price.N),
		Priced:             int32(eurOffer.Price.D),
		Price:              float64(eurOffer.Price.N) / float64(eurOffer.Price.N),
		Flags:              uint32(eurOffer.Flags),
		LastModifiedLedger: 62,
	}
	clonedQ.MockQOffers.On("GetOffersByIDs", []int64{int64(eurOffer.OfferId)}).Return([]history.Offer{mockOffer}, nil).Once()
	clonedQ.MockQOffers.On("CountOffers").Return(1, nil).Once()
	// TODO: add accounts data, trustlines and asset stats
	clonedQ.MockQData.On("CountAccountsData").Return(0, nil).Once()
	clonedQ.MockQAssetStats.On("CountTrustLines").Return(0, nil).Once()
	clonedQ.MockQAssetStats.On("GetAssetStats", "", "", db2.PageQuery{
		Order: "asc",
		Limit: assetStatsBatchSize,
	}).Return([]history.ExpAssetStat{}, nil).Once()

	next, err := verifyRangeState{
		fromLedger: 100, toLedger: 110, verifyState: true,
	}.run(s.system)
	s.Assert().NoError(err)
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
	clonedQ.AssertExpectations(s.T())
}
