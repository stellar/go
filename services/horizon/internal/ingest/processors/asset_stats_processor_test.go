//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package processors

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestAssetStatsProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(AssetStatsProcessorTestSuiteState))
}

type AssetStatsProcessorTestSuiteState struct {
	suite.Suite
	ctx       context.Context
	processor *AssetStatsProcessor
	mockQ     *history.MockQAssetStats
}

func (s *AssetStatsProcessorTestSuiteState) SetupTest() {
	s.ctx = context.Background()
	s.mockQ = &history.MockQAssetStats{}
	s.processor = NewAssetStatsProcessor(s.mockQ, "", false)
}

func (s *AssetStatsProcessorTestSuiteState) TearDownTest() {
	s.Assert().NoError(s.processor.Commit(s.ctx))
	s.mockQ.AssertExpectations(s.T())
}

func (s *AssetStatsProcessorTestSuiteState) TestCreateTrustLine() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("InsertAssetStats", s.ctx, []history.ExpAssetStat{
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: trustLineIssuer.Address(),
			AssetCode:   "EUR",
			Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
			Balances: history.ExpAssetStatBalances{
				Authorized:                      "0",
				AuthorizedToMaintainLiabilities: "0",
				Unauthorized:                    "0",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
				Contracts:                       "0",
			},
			Amount:      "0",
			NumAccounts: 1,
		},
	}, maxBatchSize).Return(nil).Once()
}

func (s *AssetStatsProcessorTestSuiteState) TestCreatePoolShareTrustLine() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset: xdr.TrustLineAsset{
			Type:            xdr.AssetTypeAssetTypePoolShare,
			LiquidityPoolId: &xdr.PoolId{1, 2, 3},
		},
		Flags: xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)
	s.Assert().NoError(s.processor.Commit(s.ctx))
	s.mockQ.AssertExpectations(s.T())
}

func (s *AssetStatsProcessorTestSuiteState) TestCreateTrustLineWithClawback() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag | xdr.TrustLineFlagsTrustlineClawbackEnabledFlag),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("InsertAssetStats", s.ctx, []history.ExpAssetStat{
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: trustLineIssuer.Address(),
			AssetCode:   "EUR",
			Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
			Balances: history.ExpAssetStatBalances{
				Authorized:                      "0",
				AuthorizedToMaintainLiabilities: "0",
				Unauthorized:                    "0",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
				Contracts:                       "0",
			},
			Amount:      "0",
			NumAccounts: 1,
		},
	}, maxBatchSize).Return(nil).Once()
}

func (s *AssetStatsProcessorTestSuiteState) TestCreateTrustLineUnauthorized() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("InsertAssetStats", s.ctx, []history.ExpAssetStat{
		{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: trustLineIssuer.Address(),
			AssetCode:   "EUR",
			Accounts:    history.ExpAssetStatAccounts{Unauthorized: 1},
			Balances: history.ExpAssetStatBalances{
				Authorized:                      "0",
				AuthorizedToMaintainLiabilities: "0",
				Unauthorized:                    "0",
				ClaimableBalances:               "0",
				LiquidityPools:                  "0",
				Contracts:                       "0",
			},
			Amount:      "0",
			NumAccounts: 0,
		},
	}, maxBatchSize).Return(nil).Once()
}

func TestAssetStatsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(AssetStatsProcessorTestSuiteLedger))
}

type AssetStatsProcessorTestSuiteLedger struct {
	suite.Suite
	ctx       context.Context
	processor *AssetStatsProcessor
	mockQ     *history.MockQAssetStats
}

func (s *AssetStatsProcessorTestSuiteLedger) SetupTest() {
	s.ctx = context.Background()
	s.mockQ = &history.MockQAssetStats{}

	s.processor = NewAssetStatsProcessor(s.mockQ, "", true)
}

func (s *AssetStatsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
}

func (s *AssetStatsProcessorTestSuiteLedger) TestInsertClaimableBalance() {
	claimableBalance := xdr.ClaimableBalanceEntry{
		Asset:  xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Amount: 12,
		BalanceId: xdr.ClaimableBalanceId{
			Type: 0,
			V0:   &xdr.Hash{1, 2, 3},
		},
	}

	nativeClaimableBalance := xdr.ClaimableBalanceEntry{
		Asset:  xdr.MustNewNativeAsset(),
		Amount: 100000000,
		BalanceId: xdr.ClaimableBalanceId{
			Type: 0,
			V0:   &xdr.Hash{1, 2, 43},
		},
	}
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	// test inserts

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &claimableBalance,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &nativeClaimableBalance,
			},
		},
	})
	s.Assert().NoError(err)

	usdClaimableBalance := xdr.ClaimableBalanceEntry{
		Asset:  xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Amount: 46,
		BalanceId: xdr.ClaimableBalanceId{
			Type: 0,
			V0:   &xdr.Hash{4, 5, 3},
		},
	}

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &usdClaimableBalance,
			},
		},
	})
	s.Assert().NoError(err)

	// test updates

	updatedClaimableBalance := claimableBalance
	updatedClaimableBalance.Amount *= 2

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &claimableBalance,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &updatedClaimableBalance,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			ClaimableBalances: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "24",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			ClaimableBalances: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "46",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestInsertTrustLine() {
	// should be ignored because it's not an trust line type
	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
	})
	s.Assert().NoError(err)

	// add trust line
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	unauthorizedTrustline := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
	}
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &unauthorizedTrustline,
			},
		},
	})
	s.Assert().NoError(err)

	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	updatedUnauthorizedTrustline := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
	}

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &updatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &unauthorizedTrustline,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &updatedUnauthorizedTrustline,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "10",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Unauthorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "10",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestInsertContractID() {
	// add trust line
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	eurID, err := trustLine.Asset.ToAsset().ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	usdID, err := xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	usdContractData, err := AssetToContractData(false, "USD", trustLineIssuer.Address(), usdID)
	s.Assert().NoError(err)

	lastModifiedLedgerSeq := xdr.Uint32(1234)
	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  usdContractData,
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("InsertAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return eurAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()

	usdAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts:    history.ExpAssetStatAccounts{},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}
	usdAssetStat.SetContractID(usdID)
	s.mockQ.On("InsertAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return usdAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestInsertContractBalance() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)
	usdID, err := xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)

	s.Assert().NoError(s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  BalanceToContractData(usdID, [32]byte{1}, 200),
		},
	}))

	usdAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Contracts: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "150",
		},
		Amount:      "0",
		NumAccounts: 0,
	}
	usdAssetStat.SetContractID(usdID)
	s.mockQ.On("GetAssetStatByContract", s.ctx, usdID).
		Return(usdAssetStat, nil).Once()

	usdAssetStat.Accounts.Contracts++
	usdAssetStat.Balances.Contracts = "350"
	s.mockQ.On("UpdateAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return usdAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveContractBalance() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)
	usdID, err := xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)

	s.Assert().NoError(s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  BalanceToContractData(usdID, [32]byte{1}, 200),
		},
	}))

	usdAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Contracts: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "200",
		},
		Amount:      "0",
		NumAccounts: 0,
	}
	usdAssetStat.SetContractID(usdID)
	s.mockQ.On("GetAssetStatByContract", s.ctx, usdID).
		Return(usdAssetStat, nil).Once()

	usdAssetStat.Accounts.Contracts = 0
	usdAssetStat.Balances.Contracts = "0"
	s.mockQ.On("UpdateAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return usdAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestInsertContractIDWithBalance() {
	// add trust line
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	eurID, err := trustLine.Asset.ToAsset().ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	usdID, err := xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	usdContractData, err := AssetToContractData(false, "USD", trustLineIssuer.Address(), usdID)
	s.Assert().NoError(err)

	lastModifiedLedgerSeq := xdr.Uint32(1234)
	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  usdContractData,
		},
	})
	s.Assert().NoError(err)

	s.Assert().NoError(s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  BalanceToContractData(usdID, [32]byte{1}, 150),
		},
	}))

	btcID := [32]byte{1, 2, 3}
	s.Assert().NoError(s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  BalanceToContractData(btcID, [32]byte{1}, 20),
		},
	}))

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("InsertAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return eurAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()

	s.mockQ.On("GetAssetStatByContract", s.ctx, btcID).
		Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()

	usdAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Contracts: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "150",
		},
		Amount:      "0",
		NumAccounts: 0,
	}
	usdAssetStat.SetContractID(usdID)
	s.mockQ.On("InsertAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return usdAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestInsertClaimableBalanceAndTrustlineAndLiquidityPool() {
	liquidityPool := xdr.LiquidityPoolEntry{
		Body: xdr.LiquidityPoolEntryBody{
			Type: xdr.LiquidityPoolTypeLiquidityPoolConstantProduct,
			ConstantProduct: &xdr.LiquidityPoolEntryConstantProduct{
				Params: xdr.LiquidityPoolConstantProductParameters{
					AssetA: xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
					AssetB: xdr.MustNewNativeAsset(),
					Fee:    20,
				},
				ReserveA:                 100,
				ReserveB:                 200,
				TotalPoolShares:          1000,
				PoolSharesTrustLineCount: 10,
			},
		},
	}

	claimableBalance := xdr.ClaimableBalanceEntry{
		Asset:  xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Amount: 12,
		BalanceId: xdr.ClaimableBalanceId{
			Type: 0,
			V0:   &xdr.Hash{1, 2, 3},
		},
	}

	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   9,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeLiquidityPool,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:          xdr.LedgerEntryTypeLiquidityPool,
				LiquidityPool: &liquidityPool,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &claimableBalance,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			ClaimableBalances: 1,
			Authorized:        1,
			LiquidityPools:    1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "9",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "12",
			LiquidityPools:                  "100",
			Contracts:                       "0",
		},
		Amount:      "9",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateContractID() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}, nil).Once()

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("UpdateAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return eurAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateContractIDWithBalance() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	s.Assert().NoError(s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  BalanceToContractData(eurID, [32]byte{1}, 150),
		},
	}))

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}, nil).Once()

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
			Contracts:  1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "150",
		},
		Amount:      "100",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("UpdateAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return eurAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateContractIDError() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	usdID, err := xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(usdID)
	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(eurAssetStat, nil).Once()

	s.Assert().EqualError(
		s.processor.Commit(s.ctx),
		"attempting to set contract id b729e13867d5c4b2d161574e00854fd41bbba3e3b0e31d36c64904414a862fa7 but row credit_alphanum4/EUR/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H already has contract id set: 6645621141097c0f88b99360ce267c3396bcfd8cfdbe9c462b0dc167b72ecdc4",
	)
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateTrustlineAndContractIDError() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	usdID, err := xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &updatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(usdID)
	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(eurAssetStat, nil).Once()

	s.Assert().EqualError(
		s.processor.Commit(s.ctx),
		"attempting to set contract id b729e13867d5c4b2d161574e00854fd41bbba3e3b0e31d36c64904414a862fa7 but row credit_alphanum4/EUR/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H already has contract id set: 6645621141097c0f88b99360ce267c3396bcfd8cfdbe9c462b0dc167b72ecdc4",
	)
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveContractIDError() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStatByContract", s.ctx, eurID).
		Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()

	s.Assert().EqualError(
		s.processor.Commit(s.ctx),
		"row for asset with contract b729e13867d5c4b2d161574e00854fd41bbba3e3b0e31d36c64904414a862fa7 is missing",
	)
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateTrustlineAndRemoveContractIDError() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &updatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}
	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(eurAssetStat, nil).Once()

	s.Assert().EqualError(
		s.processor.Commit(s.ctx),
		"row has no contract id to remove b729e13867d5c4b2d161574e00854fd41bbba3e3b0e31d36c64904414a862fa7: AssetTypeAssetTypeCreditAlphanum4 EUR GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	)
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateTrustLine() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &updatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}, nil).Once()
	s.mockQ.On("UpdateAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "110",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "110",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateTrustLineAuthorization() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	// EUR trustline: 100 unauthorized -> 10 authorized
	eurTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   100,
	}
	eurUpdatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	// USD trustline: 100 authorized -> 10 unauthorized
	usdTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   100,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	usdUpdatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
	}

	// ETH trustline: 100 authorized -> 10 authorized_to_maintain_liabilities
	ethTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("ETH", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   100,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	ethUpdatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("ETH", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedToMaintainLiabilitiesFlag),
	}

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &eurTrustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &eurUpdatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &usdTrustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &usdUpdatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &ethTrustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &ethUpdatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Unauthorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "100",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}, nil).Once()
	s.mockQ.On("UpdateAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "10",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}, nil).Once()
	s.mockQ.On("UpdateAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Unauthorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "10",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"ETH",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "ETH",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}, nil).Once()
	s.mockQ.On("UpdateAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "ETH",
		Accounts: history.ExpAssetStatAccounts{
			AuthorizedToMaintainLiabilities: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "10",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveClaimableBalance() {
	claimableBalance := xdr.ClaimableBalanceEntry{
		Asset:  xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Amount: 12,
		BalanceId: xdr.ClaimableBalanceId{
			Type: 0,
			V0:   &xdr.Hash{1, 2, 3},
		},
	}
	usdClaimableBalance := xdr.ClaimableBalanceEntry{
		Asset:  xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Amount: 21,
		BalanceId: xdr.ClaimableBalanceId{
			Type: 0,
			V0:   &xdr.Hash{4, 5, 6},
		},
	}

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &claimableBalance,
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeClaimableBalance,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:             xdr.LedgerEntryTypeClaimableBalance,
				ClaimableBalance: &usdClaimableBalance,
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			ClaimableBalances: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "12",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}, nil).Once()
	s.mockQ.On("RemoveAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Unauthorized:      1,
			ClaimableBalances: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "21",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}, nil).Once()
	s.mockQ.On("UpdateAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts:    history.ExpAssetStatAccounts{Unauthorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveTrustLine() {
	authorizedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	unauthorizedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
	}

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &authorizedTrustLine,
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &unauthorizedTrustLine,
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 1,
	}, nil).Once()
	s.mockQ.On("RemoveAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Accounts: history.ExpAssetStatAccounts{
			Unauthorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}, nil).Once()
	s.mockQ.On("RemoveAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveContractID() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("GetAssetStatByContract", s.ctx, eurID).
		Return(eurAssetStat, nil).Once()

	eurAssetStat.ContractID = nil
	s.mockQ.On("UpdateAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return eurAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateTrustlineAndRemoveContractID() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &updatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "100",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "100",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(eurAssetStat, nil).Once()

	eurAssetStat = history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Authorized: 1},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "110",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "110",
		NumAccounts: 1,
	}
	s.mockQ.On("UpdateAssetStat", s.ctx, mock.MatchedBy(func(assetStat history.ExpAssetStat) bool {
		return eurAssetStat.Equals(assetStat)
	})).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveContractIDFromZeroRow() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 0,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("GetAssetStatByContract", s.ctx, eurID).
		Return(eurAssetStat, nil).Once()

	s.mockQ.On("RemoveAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveContractIDAndBalanceZeroRow() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	s.Assert().NoError(s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  BalanceToContractData(eurID, [32]byte{1}, 9),
		},
	}))

	s.Assert().NoError(s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  BalanceToContractData(eurID, [32]byte{2}, 1),
		},
	}))

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts:    history.ExpAssetStatAccounts{Contracts: 2},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "10",
		},
		Amount:      "0",
		NumAccounts: 0,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("GetAssetStatByContract", s.ctx, eurID).
		Return(eurAssetStat, nil).Once()

	s.mockQ.On("RemoveAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveContractIDAndRow() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	eurID, err := xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ContractID("")
	s.Assert().NoError(err)
	eurContractData, err := AssetToContractData(false, "EUR", trustLineIssuer.Address(), eurID)
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeContractData,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data:                  eurContractData,
		},
	})
	s.Assert().NoError(err)

	authorizedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &authorizedTrustLine,
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "0",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "0",
		NumAccounts: 1,
	}
	eurAssetStat.SetContractID(eurID)
	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(eurAssetStat, nil).Once()

	s.mockQ.On("RemoveAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit(s.ctx))
}

func (s *AssetStatsProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	// add trust line
	lastModifiedLedgerSeq := xdr.Uint32(1234)
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err := s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
	})
	s.Assert().NoError(err)

	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()).ToTrustLineAsset(),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err = s.processor.ProcessChange(s.ctx, ingest.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &updatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat", s.ctx,
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", s.ctx, history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Accounts: history.ExpAssetStatAccounts{
			Authorized: 1,
		},
		Balances: history.ExpAssetStatBalances{
			Authorized:                      "10",
			AuthorizedToMaintainLiabilities: "0",
			Unauthorized:                    "0",
			ClaimableBalances:               "0",
			LiquidityPools:                  "0",
			Contracts:                       "0",
		},
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()
	s.Assert().NoError(s.processor.Commit(s.ctx))
}
