package processors

import (
	"context"
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

func TestAccountsSignerProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(AccountsSignerProcessorTestSuiteState))
}

type AccountsSignerProcessorTestSuiteState struct {
	suite.Suite
	processor              *DatabaseProcessor
	mockQ                  *history.MockQSigners
	mockBatchInsertBuilder *history.MockAccountSignersBatchInsertBuilder
	mockStateReader        *io.MockStateReader
	mockStateWriter        *io.MockStateWriter
}

func (s *AccountsSignerProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQSigners{}
	s.mockBatchInsertBuilder = &history.MockAccountSignersBatchInsertBuilder{}
	s.mockStateReader = &io.MockStateReader{}
	s.mockStateWriter = &io.MockStateWriter{}

	s.processor = &DatabaseProcessor{
		Action:   AccountsForSigner,
		SignersQ: s.mockQ,
	}

	// Reader and Writer should be always closed and once
	s.mockStateReader.On("Close").Return(nil).Once()
	s.mockStateWriter.On("Close").Return(nil).Once()

	s.mockQ.
		On("NewAccountSignersBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
}

func (s *AccountsSignerProcessorTestSuiteState) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
	s.mockStateReader.AssertExpectations(s.T())
	s.mockStateWriter.AssertExpectations(s.T())
}

func (s *AccountsSignerProcessorTestSuiteState) TestNoEntries() {
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

func (s *AccountsSignerProcessorTestSuiteState) TestInvalidEntry() {
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

func (s *AccountsSignerProcessorTestSuiteState) TestCreatesSigners() {
	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
						Thresholds: [4]byte{1, 1, 1, 1},
					},
				},
			},
		}, nil).Once()

	s.mockBatchInsertBuilder.
		On("Add", history.AccountSigner{
			Account: "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			Signer:  "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			Weight:  int32(1),
		}).Return(nil).Once()

	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type: xdr.LedgerEntryTypeAccount,
					Account: &xdr.AccountEntry{
						AccountId: xdr.MustAddress("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
						Signers: []xdr.Signer{
							xdr.Signer{
								Key:    xdr.MustSigner("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
								Weight: 10,
							},
						},
					},
				},
			},
		}, nil).Once()

	s.mockBatchInsertBuilder.
		On("Add", history.AccountSigner{
			Account: "GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX",
			Signer:  "GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			Weight:  int32(10),
		}).Return(nil).Once()

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

func TestAccountsSignerProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(AccountsSignerProcessorTestSuiteLedger))
}

type AccountsSignerProcessorTestSuiteLedger struct {
	suite.Suite
	processor        *DatabaseProcessor
	mockQ            *history.MockQSigners
	mockLedgerReader *io.MockLedgerReader
	mockLedgerWriter *io.MockLedgerWriter
}

func (s *AccountsSignerProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQSigners{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}

	s.processor = &DatabaseProcessor{
		Action:   AccountsForSigner,
		SignersQ: s.mockQ,
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

func (s *AccountsSignerProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *AccountsSignerProcessorTestSuiteLedger) TestNoTransactions() {
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

func (s *AccountsSignerProcessorTestSuiteLedger) TestNewAccount() {
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
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
		}, nil).Once()

	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			int32(1),
		).
		Return(int64(1), nil).Once()

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

func (s *AccountsSignerProcessorTestSuiteLedger) TestNewSigner() {
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
												Weight: 10,
											},
										},
									},
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
												Weight: 10,
											},
											xdr.Signer{
												Key:    xdr.MustSigner("GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS"),
												Weight: 15,
											},
										},
									},
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	// Remove old signer
	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
		).
		Return(int64(1), nil).Once()

	// Create new and old signer
	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
			int32(10),
		).
		Return(int64(1), nil).Once()

	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS",
			int32(15),
		).
		Return(int64(1), nil).Once()

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

func (s *AccountsSignerProcessorTestSuiteLedger) TestSignerRemoved() {
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
												Weight: 10,
											},
											xdr.Signer{
												Key:    xdr.MustSigner("GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS"),
												Weight: 15,
											},
										},
									},
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS"),
												Weight: 15,
											},
										},
									},
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	// Remove old signers
	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
		).
		Return(int64(1), nil).Once()

	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS",
		).
		Return(int64(1), nil).Once()

	// Create new signer
	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS",
			int32(15),
		).
		Return(int64(1), nil).Once()

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

// TestSignerPreAuthTxRemovedTxFailed tests if removing preauthorized transaction
// signer works even when tx failed.
func (s *AccountsSignerProcessorTestSuiteLedger) TestSignerPreAuthTxRemovedTxFailed() {
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Result: xdr.TransactionResultPair{
				Result: xdr.TransactionResult{
					Result: xdr.TransactionResultResult{
						Code: xdr.TransactionResultCodeTxFailed,
					},
				},
			},
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
												Weight: 10,
											},
											xdr.Signer{
												Key:    xdr.MustSigner("TBU2RRGLXH3E5CQHTD3ODLDF2BWDCYUSSBLLZ5GNW7JXHDIYKXZWHXL7"),
												Weight: 15,
											},
										},
									},
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
												Weight: 10,
											},
										},
									},
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	// Remove old signers
	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
		).
		Return(int64(1), nil).Once()

	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"TBU2RRGLXH3E5CQHTD3ODLDF2BWDCYUSSBLLZ5GNW7JXHDIYKXZWHXL7",
		).
		Return(int64(1), nil).Once()

	// Create new signer
	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
			int32(10),
		).
		Return(int64(1), nil).Once()

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

func (s *AccountsSignerProcessorTestSuiteLedger) TestRemoveAccount() {
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Thresholds: [4]byte{1, 1, 1, 1},
									},
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.LedgerKeyAccount{
									AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
		).
		Return(int64(1), nil).Once()

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

func (s *AccountsSignerProcessorTestSuiteLedger) TestNewAccountNoRowsAffected() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerReader.On("Close").Return(nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
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
		}, nil).Once()

	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			int32(1),
		).
		Return(int64(0), nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(verify.StateError{}, errors.Cause(err))
	s.Assert().EqualError(
		err,
		"Error in AccountsForSigner handler: No rows affected when inserting "+
			"account=GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML "+
			"signer=GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML to database",
	)
}

func (s *AccountsSignerProcessorTestSuiteLedger) TestRemoveAccountNoRowsAffected() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerReader.On("Close").Return(nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Thresholds: [4]byte{1, 1, 1, 1},
									},
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.LedgerKeyAccount{
									AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
		).
		Return(int64(0), nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(verify.StateError{}, errors.Cause(err))
	s.Assert().EqualError(
		err,
		"Error in AccountsForSigner handler: Expected "+
			"account=GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML "+
			"signer=GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML in database but not found when removing",
	)
}

func (s *AccountsSignerProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						// State
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
												Weight: 10,
											},
										},
									},
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeAccount,
									Account: &xdr.AccountEntry{
										AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										Signers: []xdr.Signer{
											xdr.Signer{
												Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
												Weight: 10,
											},
											xdr.Signer{
												Key:    xdr.MustSigner("GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS"),
												Weight: 15,
											},
										},
									},
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	// Remove old signer
	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
		).
		Return(int64(1), nil).Once()

	// Create new and old signer
	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
			int32(10),
		).
		Return(int64(1), nil).Once()

	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCAHY6JSXQFKWKP6R7U5JPXDVNV4DJWOWRFLY3Y6YPBF64QRL4BPFDNS",
			int32(15),
		).
		Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(
			io.Change{
				Type: xdr.LedgerEntryTypeAccount,
				Pre: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: 1000,
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeAccount,
						Account: &xdr.AccountEntry{
							AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
							Signers: []xdr.Signer{
								xdr.Signer{
									Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
									Weight: 10,
								},
							},
						},
					},
				},
				Post: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: 1001,
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeAccount,
						Account: &xdr.AccountEntry{
							AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
							Signers: []xdr.Signer{
								xdr.Signer{
									Key:    xdr.MustSigner("GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV"),
									Weight: 12,
								},
							},
						},
					},
				},
			}, nil).Once()

	// Update signer
	s.mockQ.
		On(
			"RemoveAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
		).
		Return(int64(1), nil).Once()

	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GCBBDQLCTNASZJ3MTKAOYEOWRGSHDFAJVI7VPZUOP7KXNHYR3HP2BUKV",
			int32(12),
		).
		Return(int64(1), nil).Once()

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

func createTransactionMeta(opMeta []xdr.OperationMeta) xdr.TransactionMeta {
	return xdr.TransactionMeta{
		V: 1,
		V1: &xdr.TransactionMetaV1{
			Operations: opMeta,
		},
	}
}
