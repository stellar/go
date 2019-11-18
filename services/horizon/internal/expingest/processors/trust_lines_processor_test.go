package processors

import (
	"context"
	"database/sql"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/verify"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

var trustLineIssuer = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")

func TestTrustLinesProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(TrustLinesProcessorTestSuiteState))
}

type TrustLinesProcessorTestSuiteState struct {
	suite.Suite
	processor              *DatabaseProcessor
	mockQ                  *history.MockQTrustLines
	mockAssetStatsQ        *history.MockQAssetStats
	mockBatchInsertBuilder *history.MockTrustLinesBatchInsertBuilder
	mockStateReader        *io.MockStateReader
	mockStateWriter        *io.MockStateWriter
}

func (s *TrustLinesProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQTrustLines{}
	s.mockAssetStatsQ = &history.MockQAssetStats{}
	s.mockBatchInsertBuilder = &history.MockTrustLinesBatchInsertBuilder{}
	s.mockStateReader = &io.MockStateReader{}
	s.mockStateWriter = &io.MockStateWriter{}

	s.processor = &DatabaseProcessor{
		Action:      TrustLines,
		TrustLinesQ: s.mockQ,
		AssetStatsQ: s.mockAssetStatsQ,
	}

	// Reader and Writer should be always closed and once
	s.mockStateReader.On("Close").Return(nil).Once()
	s.mockStateWriter.On("Close").Return(nil).Once()

	s.mockQ.
		On("NewTrustLinesBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
}

func (s *TrustLinesProcessorTestSuiteState) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockAssetStatsQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
	s.mockStateReader.AssertExpectations(s.T())
	s.mockStateWriter.AssertExpectations(s.T())
}

func (s *TrustLinesProcessorTestSuiteState) TestCreateTrustLineWithAssetStat() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	s.mockStateReader.
		On("Read").Return(
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:      xdr.LedgerEntryTypeTrustline,
					TrustLine: &trustLine,
				},
				LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			},
		},
		nil,
	).Once()

	s.mockBatchInsertBuilder.
		On("Add", trustLine, lastModifiedLedgerSeq).Return(nil).Once()

	s.mockAssetStatsQ.On("InsertAssetStats", []history.ExpAssetStat{
		history.ExpAssetStat{
			AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
			AssetIssuer: trustLineIssuer.Address(),
			AssetCode:   "EUR",
			Amount:      "0",
			NumAccounts: 1,
		},
	}, maxBatchSize).Return(nil).Once()

	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{}, stdio.EOF).Once()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	err := s.processor.ProcessState(
		context.Background(),
		&supportPipeline.Store{},
		s.mockStateReader,
		s.mockStateWriter,
	)

	s.Assert().NoError(err)
}

func (s *TrustLinesProcessorTestSuiteState) TestCreateTrustLineWithoutAssetStat() {
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	s.mockStateReader.
		On("Read").Return(
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:      xdr.LedgerEntryTypeTrustline,
					TrustLine: &trustLine,
				},
				LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			},
		},
		nil,
	).Once()

	s.mockBatchInsertBuilder.
		On("Add", trustLine, lastModifiedLedgerSeq).Return(nil).Once()

	s.mockAssetStatsQ.
		On("InsertAssetStats", []history.ExpAssetStat{}, maxBatchSize).Return(nil).Once()

	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{}, stdio.EOF).Once()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	err := s.processor.ProcessState(
		context.Background(),
		&supportPipeline.Store{},
		s.mockStateReader,
		s.mockStateWriter,
	)

	s.Assert().NoError(err)
}

func TestTrustLinesProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(TrustLinesProcessorTestSuiteLedger))
}

type TrustLinesProcessorTestSuiteLedger struct {
	suite.Suite
	processor        *DatabaseProcessor
	mockQ            *history.MockQTrustLines
	mockAssetStatsQ  *history.MockQAssetStats
	mockLedgerReader *io.MockLedgerReader
	mockLedgerWriter *io.MockLedgerWriter
}

func (s *TrustLinesProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQTrustLines{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}
	s.mockAssetStatsQ = &history.MockQAssetStats{}

	s.processor = &DatabaseProcessor{
		Action:      All,
		TrustLinesQ: s.mockQ,
		AssetStatsQ: s.mockAssetStatsQ,
	}

	// Reader and Writer should be always closed and once
	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(io.Change{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()
}

func (s *TrustLinesProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockAssetStatsQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *TrustLinesProcessorTestSuiteLedger) TestInsertTrustLine() {
	accountTransaction := io.LedgerTransaction{
		Meta: createTransactionMeta([]xdr.OperationMeta{
			xdr.OperationMeta{
				Changes: []xdr.LedgerEntryChange{
					xdr.LedgerEntryChange{
						Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
						Created: &xdr.LedgerEntry{
							Data: xdr.LedgerEntryData{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.AccountEntry{
									AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
									Thresholds: [4]byte{1, 1, 1, 1},
								},
							},
						},
					},
				},
			},
		}),
	}
	// should be ignored because it's not an trust line type
	s.Assert().NoError(
		s.processor.processLedgerTrustLines(accountTransaction.GetChanges()[0]),
	)

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
	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// Created
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
							Created: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &trustLine,
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
							Created: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &unauthorizedTrustline,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"InsertTrustLine",
		trustLine,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
	s.mockQ.On(
		"InsertTrustLine",
		unauthorizedTrustline,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockAssetStatsQ.On("InsertAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "0",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

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
	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &trustLine,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &updatedTrustLine,
								},
							},
						},
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &unauthorizedTrustline,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &updatedUnauthorizedTrustline,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockQ.On(
		"UpdateTrustLine",
		updatedTrustLine,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
	s.mockQ.On(
		"UpdateTrustLine",
		updatedUnauthorizedTrustline,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
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
	s.mockAssetStatsQ.On("UpdateAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *TrustLinesProcessorTestSuiteLedger) TestUpdateTrustLineNoRowsAffected() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerReader.On("Close").Return(nil).Once()

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
	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &trustLine,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &updatedTrustLine,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockQ.On(
		"UpdateTrustLine",
		updatedTrustLine,
		lastModifiedLedgerSeq,
	).Return(int64(0), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
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
	s.mockAssetStatsQ.On("UpdateAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(verify.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "Error in TrustLines handler: No rows affected when updating trustline: GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB credit_alphanum4/EUR/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
}

func (s *TrustLinesProcessorTestSuiteLedger) TestUpdateTrustLineAuthorization() {
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

	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &trustLine,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &updatedTrustLine,
								},
							},
						},
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &otherTrustLine,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &otherUpdatedTrustLine,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"UpdateTrustLine",
		updatedTrustLine,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockAssetStatsQ.On("InsertAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.mockQ.On(
		"UpdateTrustLine",
		otherUpdatedTrustLine,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
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
	s.mockAssetStatsQ.On("RemoveAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"USD",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}
func (s *TrustLinesProcessorTestSuiteLedger) TestRemoveTrustLine() {
	unauthorizedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		Balance:   0,
	}

	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
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
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeTrustline,
								TrustLine: &xdr.LedgerKeyTrustLine{
									AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
									Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &unauthorizedTrustLine,
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeTrustline,
								TrustLine: &xdr.LedgerKeyTrustLine{
									AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
									Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"RemoveTrustLine",
		xdr.LedgerKeyTrustLine{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		},
	).Return(int64(1), nil).Once()
	s.mockQ.On(
		"RemoveTrustLine",
		xdr.LedgerKeyTrustLine{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset("USD", trustLineIssuer.Address()),
		},
	).Return(int64(1), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
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
	s.mockAssetStatsQ.On("RemoveAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *TrustLinesProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}

	// add trust line
	trustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   0,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}
	lastModifiedLedgerSeq := xdr.Uint32(1234)
	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
							Created: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type:      xdr.LedgerEntryTypeTrustline,
									TrustLine: &trustLine,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"InsertTrustLine",
		trustLine,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()

	s.mockAssetStatsQ.On("GetAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(history.ExpAssetStat{}, sql.ErrNoRows).Once()
	s.mockAssetStatsQ.On("InsertAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "0",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	updatedTrustLine := xdr.TrustLineEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		Balance:   10,
		Flags:     xdr.Uint32(xdr.TrustLineFlagsAuthorizedFlag),
	}

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(
			io.Change{
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
			}, nil).Once()

	s.mockQ.On(
		"UpdateTrustLine",
		updatedTrustLine,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
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
	s.mockAssetStatsQ.On("UpdateAssetStat", history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: trustLineIssuer.Address(),
		AssetCode:   "EUR",
		Amount:      "10",
		NumAccounts: 1,
	}).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(io.Change{}, stdio.EOF).Once()

	s.mockLedgerReader.On("Close").Return(nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *TrustLinesProcessorTestSuiteLedger) TestRemoveTrustlineNoRowsAffected() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerReader.On("Close").Return(nil).Once()

	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
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
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeTrustline,
								TrustLine: &xdr.LedgerKeyTrustLine{
									AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
									Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"RemoveTrustLine",
		xdr.LedgerKeyTrustLine{
			AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
			Asset:     xdr.MustNewCreditAsset("EUR", trustLineIssuer.Address()),
		},
	).Return(int64(0), nil).Once()
	s.mockAssetStatsQ.On("GetAssetStat",
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
	s.mockAssetStatsQ.On("RemoveAssetStat",
		xdr.AssetTypeAssetTypeCreditAlphanum4,
		"EUR",
		trustLineIssuer.Address(),
	).Return(int64(1), nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(verify.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "Error in TrustLines handler: No rows affected when removing trustline: GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB credit_alphanum4/EUR/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
}
