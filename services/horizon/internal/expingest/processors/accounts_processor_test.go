package processors

import (
	"context"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestAccountsProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(AccountsProcessorTestSuiteState))
}

type AccountsProcessorTestSuiteState struct {
	suite.Suite
	processor *AccountsProcessor
	mockQ     *history.MockQAccounts
}

func (s *AccountsProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQAccounts{}

	s.processor = NewAccountsProcessor(s.mockQ)
}

func (s *AccountsProcessorTestSuiteState) TearDownTest() {
	s.Assert().NoError(s.processor.Commit())
	s.mockQ.AssertExpectations(s.T())
}

func (s *AccountsProcessorTestSuiteState) TestNoEntries() {
	// Nothing processed, assertions in TearDownTest.
}

func (s *AccountsProcessorTestSuiteState) TestCreatesAccounts() {
	account := xdr.AccountEntry{
		AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Thresholds: [4]byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockQ.On(
		"UpsertAccounts",
		[]xdr.LedgerEntry{
			xdr.LedgerEntry{
				LastModifiedLedgerSeq: lastModifiedLedgerSeq,
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &account,
				},
			},
		},
	).Return(nil).Once()

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &account,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)
}

func TestAccountsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(AccountsProcessorTestSuiteLedger))
}

type AccountsProcessorTestSuiteLedger struct {
	suite.Suite
	context   context.Context
	processor *AccountsProcessor
	mockQ     *history.MockQAccounts
}

func (s *AccountsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQAccounts{}

	s.processor = NewAccountsProcessor(s.mockQ)
}

func (s *AccountsProcessorTestSuiteLedger) TearDownTest() {
	s.Assert().NoError(s.processor.Commit())
	s.mockQ.AssertExpectations(s.T())
}

func (s *AccountsProcessorTestSuiteLedger) TestNoTransactions() {
	// Nothing processed, assertions in TearDownTest.
}

func (s *AccountsProcessorTestSuiteLedger) TestNewAccount() {
	account := xdr.AccountEntry{
		AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Thresholds: [4]byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &account,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)

	updatedAccount := xdr.AccountEntry{
		AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Thresholds: [4]byte{0, 1, 2, 3},
		HomeDomain: "stellar.org",
	}

	err = s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &account,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &updatedAccount,
			},
		},
	})
	s.Assert().NoError(err)

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockQ.On(
		"UpsertAccounts",
		[]xdr.LedgerEntry{
			xdr.LedgerEntry{
				LastModifiedLedgerSeq: lastModifiedLedgerSeq,
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &updatedAccount,
				},
			},
		},
	).Return(nil).Once()
}

func (s *AccountsProcessorTestSuiteLedger) TestRemoveAccount() {
	s.mockQ.On(
		"RemoveAccount",
		"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
	).Return(int64(1), nil).Once()

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)
}

func (s *AccountsProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	account := xdr.AccountEntry{
		AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Thresholds: [4]byte{1, 1, 1, 1},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &account,
			},
		},
	})
	s.Assert().NoError(err)

	updatedAccount := xdr.AccountEntry{
		AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		Thresholds: [4]byte{0, 1, 2, 3},
		HomeDomain: "stellar.org",
	}

	err = s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &account,
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq + 1,
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: &updatedAccount,
			},
		},
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"UpsertAccounts",
		[]xdr.LedgerEntry{
			xdr.LedgerEntry{
				LastModifiedLedgerSeq: lastModifiedLedgerSeq + 1,
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &updatedAccount,
				},
			},
		},
	).Return(nil).Once()
}

func (s *AccountsProcessorTestSuiteLedger) TestFeeProcessedBeforeEverythingElse() {
	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
					Balance:   200,
				},
			},
		},
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
					Balance:   100,
				},
			},
		},
	})
	s.Assert().NoError(err)

	err = s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
					Balance:   100,
				},
			},
		},
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
					Balance:   300,
				},
			},
		},
	})
	s.Assert().NoError(err)

	expectedAccount := xdr.AccountEntry{
		AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
		Balance:   300,
	}

	s.mockQ.On(
		"UpsertAccounts",
		[]xdr.LedgerEntry{
			xdr.LedgerEntry{
				LastModifiedLedgerSeq: 0,
				Data: xdr.LedgerEntryData{
					Type:    xdr.LedgerEntryTypeAccount,
					Account: &expectedAccount,
				},
			},
		},
	).Return(nil).Once()
}
