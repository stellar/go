package processors

import (
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestAccountsDataProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(AccountsDataProcessorTestSuiteState))
}

type AccountsDataProcessorTestSuiteState struct {
	suite.Suite
	processor              *AccountDataProcessor
	mockQ                  *history.MockQData
	mockBatchInsertBuilder *history.MockAccountDataBatchInsertBuilder
}

func (s *AccountsDataProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQData{}
	s.mockBatchInsertBuilder = &history.MockAccountDataBatchInsertBuilder{}

	s.mockQ.
		On("NewAccountDataBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.processor = NewAccountDataProcessor(s.mockQ)
}

func (s *AccountsDataProcessorTestSuiteState) TearDownTest() {
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.Assert().NoError(s.processor.Commit())

	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *AccountsDataProcessorTestSuiteState) TestNoEntries() {
	// Nothing processed, assertions in TearDownTest.
}

func (s *AccountsDataProcessorTestSuiteState) TestCreatesAccounts() {
	data := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	s.mockBatchInsertBuilder.On("Add", data, lastModifiedLedgerSeq).Return(nil).Once()

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeData,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &data,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)
}

func TestAccountsDataProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(AccountsDataProcessorTestSuiteLedger))
}

type AccountsDataProcessorTestSuiteLedger struct {
	suite.Suite
	processor              *AccountDataProcessor
	mockQ                  *history.MockQData
	mockBatchInsertBuilder *history.MockAccountDataBatchInsertBuilder
}

func (s *AccountsDataProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQData{}
	s.mockBatchInsertBuilder = &history.MockAccountDataBatchInsertBuilder{}

	s.mockQ.
		On("NewAccountDataBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.processor = NewAccountDataProcessor(s.mockQ)
}

func (s *AccountsDataProcessorTestSuiteLedger) TearDownTest() {
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.Assert().NoError(s.processor.Commit())
	s.mockQ.AssertExpectations(s.T())
}

func (s *AccountsDataProcessorTestSuiteLedger) TestNoTransactions() {
	// Nothing processed, assertions in TearDownTest.
}

func (s *AccountsDataProcessorTestSuiteLedger) TestNewAccountData() {
	data := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeData,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &data,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)

	updatedData := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{2, 2, 2, 2},
	}

	err = s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeData,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &data,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
		},
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &updatedData,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockBatchInsertBuilder.On(
		"Add",
		updatedData,
		lastModifiedLedgerSeq,
	).Return(nil).Once()
}

func (s *AccountsDataProcessorTestSuiteLedger) TestUpdateAccountData() {
	data := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	updatedData := xdr.DataEntry{
		AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		DataName:  "test",
		DataValue: []byte{2, 2, 2, 2},
	}

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeData,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &data,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
		},
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &updatedData,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"UpdateAccountData",
		updatedData,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()
}

func (s *AccountsDataProcessorTestSuiteLedger) TestRemoveAccountData() {
	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeData,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &xdr.DataEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					DataName:  "test",
					DataValue: []byte{1, 1, 1, 1},
				},
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"RemoveAccountData",
		xdr.LedgerKeyData{
			AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			DataName:  "test",
		},
	).Return(int64(1), nil).Once()
}
