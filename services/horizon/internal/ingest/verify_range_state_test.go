//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"testing"

	"github.com/guregu/null"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func TestVerifyRangeStateTestSuite(t *testing.T) {
	suite.Run(t, new(VerifyRangeStateTestSuite))
}

type VerifyRangeStateTestSuite struct {
	suite.Suite
	ledgerBackend  *ledgerbackend.MockDatabaseBackend
	historyQ       *mockDBQ
	historyAdapter *mockHistoryArchiveAdapter
	runner         *mockProcessorsRunner
	system         *system
}

func (s *VerifyRangeStateTestSuite) SetupTest() {
	s.ledgerBackend = &ledgerbackend.MockDatabaseBackend{}
	s.historyQ = &mockDBQ{}
	s.historyAdapter = &mockHistoryArchiveAdapter{}
	s.runner = &mockProcessorsRunner{}
	s.system = &system{
		ctx:               context.Background(),
		historyQ:          s.historyQ,
		historyAdapter:    s.historyAdapter,
		ledgerBackend:     s.ledgerBackend,
		runner:            s.runner,
		checkpointManager: historyarchive.NewCheckpointManager(64),
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

func (s *VerifyRangeStateTestSuite) TestGetLastLedgerIngestReturnsError() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerIngest").Return(uint32(0), errors.New("my error")).Once()

	next, err := verifyRangeState{fromLedger: 100, toLedger: 200}.run(s.system)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "Error getting last ingested ledger: my error")
	s.Assert().Equal(
		transition{node: stopState{}, sleepDuration: 0},
		next,
	)
}

func (s *VerifyRangeStateTestSuite) TestGetLastLedgerIngestNonEmpty() {
	s.historyQ.On("Begin").Return(nil).Once()
	s.historyQ.On("GetLastLedgerIngest").Return(uint32(100), nil).Once()

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
	s.historyQ.On("GetLastLedgerIngest").Return(uint32(0), nil).Once()
	s.ledgerBackend.On("PrepareRange", ledgerbackend.BoundedRange(100, 200)).Return(nil).Once()

	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(ingest.StatsChangeProcessorResults{}, errors.New("my error")).Once()

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
	s.historyQ.On("GetLastLedgerIngest").Return(uint32(0), nil).Once()
	s.ledgerBackend.On("PrepareRange", ledgerbackend.BoundedRange(100, 200)).Return(nil).Once()
	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(ingest.StatsChangeProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", uint32(100)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	for i := uint32(101); i <= 200; i++ {
		s.historyQ.On("Begin").Return(nil).Once()
		s.runner.On("RunAllProcessorsOnLedger", i).Return(
			ingest.StatsChangeProcessorResults{},
			processorsRunDurations{},
			processors.StatsLedgerTransactionProcessorResults{},
			processorsRunDurations{},
			nil,
		).Once()
		s.historyQ.On("UpdateLastLedgerIngest", i).Return(nil).Once()
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
	s.historyQ.On("GetLastLedgerIngest").Return(uint32(0), nil).Once()
	s.ledgerBackend.On("PrepareRange", ledgerbackend.BoundedRange(100, 110)).Return(nil).Once()
	s.runner.On("RunHistoryArchiveIngestion", uint32(100)).Return(ingest.StatsChangeProcessorResults{}, nil).Once()
	s.historyQ.On("UpdateLastLedgerIngest", uint32(100)).Return(nil).Once()
	s.historyQ.On("Commit").Return(nil).Once()

	for i := uint32(101); i <= 110; i++ {
		s.historyQ.On("Begin").Return(nil).Once()
		s.runner.On("RunAllProcessorsOnLedger", i).Return(
			ingest.StatsChangeProcessorResults{},
			processorsRunDurations{},
			processors.StatsLedgerTransactionProcessorResults{},
			processorsRunDurations{},
			nil,
		).Once()
		s.historyQ.On("UpdateLastLedgerIngest", i).Return(nil).Once()
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
	clonedQ.On("GetLastLedgerIngestNonBlocking").Return(uint32(63), nil).Once()
	mockChangeReader := &ingest.MockChangeReader{}
	mockChangeReader.On("Close").Return(nil).Once()
	mockAccountID := "GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX"
	sponsor := "GAREDQSXC7QZYJLVMTU7XZW4LSILQ4M5U4GNLO523LEWZ3JBRC5E4HLE"
	signers := []string{
		"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
		"GA25GQLHJU3LPEJXEIAXK23AWEA5GWDUGRSHTQHDFT6HXHVMRULSQJUJ",
		"GC6G3EQFKOKIIZFTJQSCHTSXBVC4XO3I64F5IBRQNS3E5SW3MO3KWGMT",
	}
	accountChange := ingest.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:  xdr.MustAddress(mockAccountID),
					Balance:    xdr.Int64(600),
					Thresholds: [4]byte{1, 0, 0, 0},
					Signers: []xdr.Signer{
						{
							Key:    xdr.MustSigner(signers[0]),
							Weight: 1,
						},
						{
							Key:    xdr.MustSigner(signers[1]),
							Weight: 2,
						},
						{
							Key:    xdr.MustSigner(signers[2]),
							Weight: 3,
						},
					},
					Ext: xdr.AccountEntryExt{
						V: 1,
						V1: &xdr.AccountEntryExtensionV1{
							Liabilities: xdr.Liabilities{
								Buying:  1,
								Selling: 1,
							},
							Ext: xdr.AccountEntryExtensionV1Ext{
								V: 2,
								V2: &xdr.AccountEntryExtensionV2{
									NumSponsored:  xdr.Uint32(0),
									NumSponsoring: xdr.Uint32(2),
									SignerSponsoringIDs: []xdr.SponsorshipDescriptor{
										nil,
										xdr.MustAddressPtr(mockAccountID),
										xdr.MustAddressPtr(sponsor),
									},
								},
							},
						},
					},
				},
			},
			LastModifiedLedgerSeq: xdr.Uint32(62),
			Ext: xdr.LedgerEntryExt{
				V: 1,
				V1: &xdr.LedgerEntryExtensionV1{
					SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}
	offerChange := ingest.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:  xdr.LedgerEntryTypeOffer,
				Offer: &eurOffer,
			},
			LastModifiedLedgerSeq: xdr.Uint32(62),
			Ext: xdr.LedgerEntryExt{
				V: 1,
				V1: &xdr.LedgerEntryExtensionV1{
					SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}
	claimableBalance := history.ClaimableBalance{
		BalanceID: xdr.ClaimableBalanceId{
			Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
			V0:   &xdr.Hash{1, 2, 3},
		},
		Asset:              xdr.MustNewNativeAsset(),
		Amount:             10,
		LastModifiedLedger: 62,
		Claimants: []history.Claimant{
			{
				Destination: mockAccountID,
				Predicate: xdr.ClaimPredicate{
					Type: xdr.ClaimPredicateTypeClaimPredicateUnconditional,
				},
			},
		},
		Sponsor: null.StringFrom("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Flags:   uint32(xdr.ClaimableBalanceFlagsClaimableBalanceClawbackEnabledFlag),
	}
	claimableBalanceChange := ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &xdr.ClaimableBalanceEntry{
					BalanceId: claimableBalance.BalanceID,
					Claimants: []xdr.Claimant{
						{
							Type: xdr.ClaimantTypeClaimantTypeV0,
							V0: &xdr.ClaimantV0{
								Destination: xdr.MustAddress(claimableBalance.Claimants[0].Destination),
								Predicate:   claimableBalance.Claimants[0].Predicate,
							},
						},
					},
					Asset:  claimableBalance.Asset,
					Amount: claimableBalance.Amount,
					Ext: xdr.ClaimableBalanceEntryExt{
						V: 1,
						V1: &xdr.ClaimableBalanceEntryExtensionV1{
							Flags: xdr.Uint32(xdr.ClaimableBalanceFlagsClaimableBalanceClawbackEnabledFlag),
						},
					},
				},
			},
			LastModifiedLedgerSeq: xdr.Uint32(62),
			Ext: xdr.LedgerEntryExt{
				V: 1,
				V1: &xdr.LedgerEntryExtensionV1{
					SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}

	mockChangeReader.On("Read").Return(accountChange, nil).Once()
	mockChangeReader.On("Read").Return(offerChange, nil).Once()
	mockChangeReader.On("Read").Return(claimableBalanceChange, nil).Once()
	mockChangeReader.On("Read").Return(ingest.Change{}, io.EOF).Once()
	mockChangeReader.On("Read").Return(ingest.Change{}, io.EOF).Once()
	s.historyAdapter.On("GetState", mock.AnythingOfType("*context.emptyCtx"), uint32(63)).Return(mockChangeReader, nil).Once()
	mockAccount := history.AccountEntry{
		AccountID:          mockAccountID,
		Balance:            600,
		LastModifiedLedger: 62,
		MasterWeight:       1,
		NumSponsored:       0,
		NumSponsoring:      2,
		BuyingLiabilities:  1,
		SellingLiabilities: 1,
		Sponsor:            null.StringFrom("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
	}

	clonedQ.MockQAccounts.On("GetAccountsByIDs", []string{mockAccountID}).Return([]history.AccountEntry{mockAccount}, nil).Once()
	clonedQ.MockQSigners.On("SignersForAccounts", []string{mockAccountID}).Return([]history.AccountSigner{
		{
			Account: mockAccountID,
			Signer:  mockAccountID,
			Weight:  1,
		},
		{
			Account: mockAccountID,
			Signer:  signers[0],
			Weight:  1,
		},
		{
			Account: mockAccountID,
			Signer:  signers[2],
			Weight:  3,
			Sponsor: null.StringFrom(sponsor),
		},
		{
			Account: mockAccountID,
			Signer:  signers[1],
			Weight:  2,
			Sponsor: null.StringFrom(mockAccountID),
		},
	}, nil).Once()
	clonedQ.MockQSigners.On("CountAccounts").Return(1, nil).Once()
	mockOffer := history.Offer{
		SellerID:           eurOffer.SellerId.Address(),
		OfferID:            int64(eurOffer.OfferId),
		SellingAsset:       eurOffer.Selling,
		BuyingAsset:        eurOffer.Buying,
		Amount:             int64(eurOffer.Amount),
		Pricen:             int32(eurOffer.Price.N),
		Priced:             int32(eurOffer.Price.D),
		Price:              float64(eurOffer.Price.N) / float64(eurOffer.Price.N),
		Flags:              uint32(eurOffer.Flags),
		LastModifiedLedger: 62,
		Sponsor:            null.StringFrom("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
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

	clonedQ.MockQClaimableBalances.On("CountClaimableBalances").Return(1, nil).Once()
	clonedQ.MockQClaimableBalances.
		On("GetClaimableBalancesByID", []xdr.ClaimableBalanceId{claimableBalanceChange.Post.Data.ClaimableBalance.BalanceId}).
		Return([]history.ClaimableBalance{claimableBalance}, nil).Once()

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

func (s *VerifyRangeStateTestSuite) TestVerifyFailsWhenAssetStatsMismatch() {
	set := processors.AssetStatSet{}

	trustLineIssuer := xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	set.AddTrustline(
		ingest.Change{
			Post: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					TrustLine: &xdr.TrustLineEntry{
						AccountId: xdr.MustAddress(keypair.MustRandom().Address()),
						Balance:   123,
						Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
						Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedToMaintainLiabilitiesFlag),
					},
				},
			},
		},
	)

	stat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetCode:   "EUR",
		AssetIssuer: trustLineIssuer.Address(),
		Accounts: history.ExpAssetStatAccounts{
			Unauthorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "123",
		},
		Amount:      "0",
		NumAccounts: 0,
	}

	s.historyQ.MockQAssetStats.On("GetAssetStats", "", "", db2.PageQuery{
		Order: "asc",
		Limit: assetStatsBatchSize,
	}).Return([]history.ExpAssetStat{stat}, nil).Once()
	s.historyQ.MockQAssetStats.On("GetAssetStats", "", "", db2.PageQuery{
		Cursor: stat.PagingToken(),
		Order:  "asc",
		Limit:  assetStatsBatchSize,
	}).Return([]history.ExpAssetStat{}, nil).Once()

	err := checkAssetStats(set, s.historyQ)
	s.Assert().EqualError(err, fmt.Sprintf("db asset stat with code EUR issuer %s does not match asset stat from HAS", trustLineIssuer.Address()))

	// Satisfy the mock
	s.historyQ.Rollback()
}
