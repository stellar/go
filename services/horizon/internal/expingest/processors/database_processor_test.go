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

func TestDatabaseProcessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(DatabaseProcessorTestSuiteState))
}

type DatabaseProcessorTestSuiteState struct {
	suite.Suite
	processor       *DatabaseProcessor
	mockQ           *history.MockQSigners
	mockStateReader *io.MockStateReader
	mockStateWriter *io.MockStateWriter
}

func (s *DatabaseProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQSigners{}
	s.mockStateReader = &io.MockStateReader{}
	s.mockStateWriter = &io.MockStateWriter{}

	s.processor = &DatabaseProcessor{
		Action:   AccountsForSigner,
		HistoryQ: s.mockQ,
	}

	// Reader and Writer should be always closed and once
	s.mockStateReader.
		On("Close").
		Return(nil).Once()

	s.mockStateWriter.
		On("Close").
		Return(nil).Once()
}

func (s *DatabaseProcessorTestSuiteState) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockStateReader.AssertExpectations(s.T())
	s.mockStateWriter.AssertExpectations(s.T())
}

func (s *DatabaseProcessorTestSuiteState) TestNoEntries() {
	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{}, stdio.EOF).Once()

	err := s.processor.ProcessState(
		context.Background(),
		&supportPipeline.Store{},
		s.mockStateReader,
		s.mockStateWriter,
	)

	s.Assert().NoError(err)
}

func (s *DatabaseProcessorTestSuiteState) TestInvalidEntry() {
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

func (s *DatabaseProcessorTestSuiteState) TestCreatesSigners() {
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

	s.mockQ.
		On(
			"CreateAccountSigner",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			int32(1),
		).
		Return(nil).Once()

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

	s.mockQ.
		On(
			"CreateAccountSigner",
			"GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX",
			"GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML",
			int32(10),
		).
		Return(nil).Once()

	s.mockStateReader.
		On("Read").
		Return(xdr.LedgerEntryChange{}, stdio.EOF).Once()

	err := s.processor.ProcessState(
		context.Background(),
		&supportPipeline.Store{},
		s.mockStateReader,
		s.mockStateWriter,
	)

	s.Assert().NoError(err)
}
