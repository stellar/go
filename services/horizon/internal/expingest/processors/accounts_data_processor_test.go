package processors

import (
	"context"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestAccountsDataProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(AccountsDataProcessorTestSuiteState))
}

type AccountsDataProcessorTestSuiteState struct {
	suite.Suite
	processor              *DatabaseProcessor
	mockQ                  *history.MockQData
	mockBatchInsertBuilder *history.MockAccountDataBatchInsertBuilder
	mockStateReader        *io.MockStateReader
	mockStateWriter        *io.MockStateWriter
}

func (s *AccountsDataProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQData{}
	s.mockBatchInsertBuilder = &history.MockAccountDataBatchInsertBuilder{}
	s.mockStateReader = &io.MockStateReader{}
	s.mockStateWriter = &io.MockStateWriter{}

	s.processor = &DatabaseProcessor{
		Action: Data,
		DataQ:  s.mockQ,
	}

	// Reader and Writer should be always closed and once
	s.mockStateReader.On("Close").Return(nil).Once()
	s.mockStateWriter.On("Close").Return(nil).Once()

	s.mockQ.
		On("NewAccountDataBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
}

func (s *AccountsDataProcessorTestSuiteState) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
	s.mockStateReader.AssertExpectations(s.T())
	s.mockStateWriter.AssertExpectations(s.T())
}

func (s *AccountsDataProcessorTestSuiteState) TestNoEntries() {
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

func (s *AccountsDataProcessorTestSuiteState) TestInvalidEntry() {
	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
		}, nil).Once()

	err := s.processor.ProcessState(
		context.Background(),
		&supportPipeline.Store{},
		s.mockStateReader,
		s.mockStateWriter,
	)

	s.Assert().EqualError(err, "DatabaseProcessor requires LedgerEntryChangeTypeLedgerEntryState changes only")
}

func (s *AccountsDataProcessorTestSuiteState) TestCreatesAccounts() {
	data := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeData,
					Data: &data,
				},
				LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			},
		}, nil).Once()

	s.mockBatchInsertBuilder.On("Add", data, lastModifiedLedgerSeq).Return(nil).Once()

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

func TestAccountsDataProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(AccountsDataProcessorTestSuiteLedger))
}

type AccountsDataProcessorTestSuiteLedger struct {
	suite.Suite
	processor        *DatabaseProcessor
	mockQ            *history.MockQData
	mockLedgerReader *io.MockLedgerReader
	mockLedgerWriter *io.MockLedgerWriter
}

func (s *AccountsDataProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQData{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}

	s.processor = &DatabaseProcessor{
		Action: Data,
		DataQ:  s.mockQ,
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

func (s *AccountsDataProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *AccountsDataProcessorTestSuiteLedger) TestNoTransactions() {
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

func (s *AccountsDataProcessorTestSuiteLedger) TestNewAccount() {
	data := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
							Created: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeData,
									Data: &data,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"InsertAccountData",
		data,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()

	updatedData := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{2, 2, 2, 2},
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
									Type: xdr.LedgerEntryTypeData,
									Data: &data,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeData,
									Data: &updatedData,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockQ.On(
		"UpdateAccountData",
		updatedData,
		lastModifiedLedgerSeq,
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

func (s *AccountsDataProcessorTestSuiteLedger) TestRemoveAccount() {
	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeData,
									Data: &xdr.DataEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										DataName:  "test",
										DataValue: []byte{1, 1, 1, 1},
									},
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeData,
								Data: &xdr.LedgerKeyData{
									AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
									DataName:  "test",
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"RemoveAccountData",
		xdr.LedgerKeyData{
			AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			DataName:  "test",
		},
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

func (s *AccountsDataProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}

	data := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
							Created: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeData,
									Data: &data,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.On(
		"InsertAccountData",
		data,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	// Process ledger entry upgrades
	modifiedData := data
	modifiedData.DataValue = []byte{2, 2, 2, 2}

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(
			io.Change{
				Type: xdr.LedgerEntryTypeData,
				Pre: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: lastModifiedLedgerSeq,
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeData,
						Data: &data,
					},
				},
				Post: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: lastModifiedLedgerSeq + 1,
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeData,
						Data: &modifiedData,
					},
				},
			}, nil).Once()

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(io.Change{}, stdio.EOF).Once()

	s.mockLedgerReader.On("Close").Return(nil).Once()

	s.mockQ.On(
		"UpdateAccountData",
		modifiedData,
		lastModifiedLedgerSeq+1,
	).Return(int64(1), nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}
