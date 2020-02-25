package processors

import (
	"database/sql"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestAssetStatsProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(AssetStatsProcessorTestSuiteState))
}

type AssetStatsProcessorTestSuiteState struct {
	suite.Suite
	processor *AssetStatsProcessor
	mockQ     *history.MockQAssetStats
}

func (s *AssetStatsProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQAssetStats{}
	s.processor = NewAssetStatsProcessor(s.mockQ, false)
}

func (s *AssetStatsProcessorTestSuiteState) TearDownTest() {
	s.Assert().NoError(s.processor.Commit())
	s.mockQ.AssertExpectations(s.T())
}

func (s *AssetStatsProcessorTestSuiteState) TestCreateTrustLine() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	err := s.processor.ProcessChange(io.Change{
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

	s.mockQ.On("InsertAssetStats", []history.ExpAssetStat{
		history.ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: trustLineIssuer.Address(),
			AssetCode:   "EUR",
			Amount:      "0",
			NumAccounts: 1,
		},
	}, maxBatchSize).Return(nil).Once()
}

func (s *AssetStatsProcessorTestSuiteState) TestCreateTrustLineUnauthorized() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	err := s.processor.ProcessChange(io.Change{
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

	s.mockQ.
		On("InsertAssetStats", []history.ExpAssetStat{}, maxBatchSize).Return(nil).Once()
}

func TestAssetStatsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(AssetStatsProcessorTestSuiteLedger))
}

type AssetStatsProcessorTestSuiteLedger struct {
	suite.Suite
	processor *AssetStatsProcessor
	mockQ     *history.MockQAssetStats
}

func (s *AssetStatsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQAssetStats{}

	s.processor = NewAssetStatsProcessor(s.mockQ, true)
}

func (s *AssetStatsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
}

func (s *AssetStatsProcessorTestSuiteLedger) TestInsertTrustLine() {
	// should be ignored because it's not an trust line type
	err := s.processor.ProcessChange(io.Change{
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
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	unauthorizedTrustline := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Balance:   0,
	}
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	err = s.processor.ProcessChange(io.Change{
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

	err = s.processor.ProcessChange(io.Change{
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
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	updatedUnauthorizedTrustline := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Balance:   10,
	}

	err = s.processor.ProcessChange(io.Change{
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

	err = s.processor.ProcessChange(io.Change{
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

	s.mockQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit())
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateTrustLine() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err := s.processor.ProcessChange(io.Change{
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

	s.mockQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "100",
		NumAccounts: 1,
	}, nil).Once()
	s.mockQ.On("UpdateAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "110",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.Assert().NoError(s.processor.Commit())
}

func (s *AssetStatsProcessorTestSuiteLedger) TestUpdateTrustLineAuthorization() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   100,
	}
	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	otherTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Balance:   100,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	otherUpdatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Balance:   10,
	}

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
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

	err = s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &otherTrustLine,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &otherUpdatedTrustLine,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.mockQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "USD",
		Amount:      "100",
		NumAccounts: 1,
	}, nil).Once()
	s.mockQ.On("RemoveAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()
	s.Assert().NoError(s.processor.Commit())
}

func (s *AssetStatsProcessorTestSuiteLedger) TestRemoveTrustLine() {
	unauthorizedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Balance:   0,
	}

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeTrustline,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeTrustline,
				TrustLine: &xdr.TrustLineEntry{
					AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
					Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
					Balance:   0,
					Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
				},
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(io.Change{
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

	s.mockQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "0",
		NumAccounts: 1,
	}, nil).Once()
	s.mockQ.On("RemoveAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()
	s.Assert().NoError(s.processor.Commit())
}

func (s *AssetStatsProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	// add trust line
	lastModifiedLedgerSeq := xdr.Uint32(1234)
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err := s.processor.ProcessChange(io.Change{
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
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	err = s.processor.ProcessChange(io.Change{
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

	s.mockQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockQ.On("InsertAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()
	s.Assert().NoError(s.processor.Commit())
}
