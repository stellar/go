package expingest

import (
	"errors"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var _ horizonChangeProcessor = (*mockHorizonChangeProcessor)(nil)

type mockHorizonChangeProcessor struct {
	mock.Mock
}

func (m *mockHorizonChangeProcessor) ProcessChange(change io.Change) error {
	args := m.Called(change)
	return args.Error(0)
}

func (m *mockHorizonChangeProcessor) Commit() error {
	args := m.Called()
	return args.Error(0)
}

var _ horizonTransactionProcessor = (*mockHorizonTransactionProcessor)(nil)

type mockHorizonTransactionProcessor struct {
	mock.Mock
}

func (m *mockHorizonTransactionProcessor) ProcessTransaction(transaction io.LedgerTransaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}

func (m *mockHorizonTransactionProcessor) Commit() error {
	args := m.Called()
	return args.Error(0)
}

type GroupChangeProcessorsTestSuiteLedger struct {
	suite.Suite
	processors *groupChangeProcessors
	processorA *mockHorizonChangeProcessor
	processorB *mockHorizonChangeProcessor
}

func TestGroupChangeProcessorsTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(GroupChangeProcessorsTestSuiteLedger))
}

func (s *GroupChangeProcessorsTestSuiteLedger) SetupTest() {
	s.processorA = &mockHorizonChangeProcessor{}
	s.processorB = &mockHorizonChangeProcessor{}
	s.processors = &groupChangeProcessors{
		s.processorA,
		s.processorB,
	}
}

func (s *GroupChangeProcessorsTestSuiteLedger) TearDownTest() {
	s.processorA.AssertExpectations(s.T())
	s.processorB.AssertExpectations(s.T())
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestProcessChangeFails() {
	change := io.Change{}
	s.processorA.
		On("ProcessChange", change).
		Return(errors.New("transient error")).Once()

	err := s.processors.ProcessChange(change)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *expingest.mockHorizonChangeProcessor.ProcessChange: transient error")
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestProcessChangeSucceeds() {
	change := io.Change{}
	s.processorA.
		On("ProcessChange", change).
		Return(nil).Once()
	s.processorB.
		On("ProcessChange", change).
		Return(nil).Once()

	err := s.processors.ProcessChange(change)
	s.Assert().NoError(err)
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestCommitFails() {
	s.processorA.
		On("Commit").
		Return(errors.New("transient error")).Once()

	err := s.processors.Commit()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *expingest.mockHorizonChangeProcessor.Commit: transient error")
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestCommitSucceeds() {
	s.processorA.
		On("Commit").
		Return(nil).Once()
	s.processorB.
		On("Commit").
		Return(nil).Once()

	err := s.processors.Commit()
	s.Assert().NoError(err)
}

type GroupTransactionProcessorsTestSuiteLedger struct {
	suite.Suite
	processors *groupTransactionProcessors
	processorA *mockHorizonTransactionProcessor
	processorB *mockHorizonTransactionProcessor
}

func TestGroupTransactionProcessorsTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(GroupTransactionProcessorsTestSuiteLedger))
}

func (s *GroupTransactionProcessorsTestSuiteLedger) SetupTest() {
	s.processorA = &mockHorizonTransactionProcessor{}
	s.processorB = &mockHorizonTransactionProcessor{}
	s.processors = &groupTransactionProcessors{
		s.processorA,
		s.processorB,
	}
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TearDownTest() {
	s.processorA.AssertExpectations(s.T())
	s.processorB.AssertExpectations(s.T())
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestProcessTransactionFails() {
	transaction := io.LedgerTransaction{}
	s.processorA.
		On("ProcessTransaction", transaction).
		Return(errors.New("transient error")).Once()

	err := s.processors.ProcessTransaction(transaction)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *expingest.mockHorizonTransactionProcessor.ProcessTransaction: transient error")
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestProcessTransactionSucceeds() {
	transaction := io.LedgerTransaction{}
	s.processorA.
		On("ProcessTransaction", transaction).
		Return(nil).Once()
	s.processorB.
		On("ProcessTransaction", transaction).
		Return(nil).Once()

	err := s.processors.ProcessTransaction(transaction)
	s.Assert().NoError(err)
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestCommitFails() {
	s.processorA.
		On("Commit").
		Return(errors.New("transient error")).Once()

	err := s.processors.Commit()
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *expingest.mockHorizonTransactionProcessor.Commit: transient error")
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestCommitSucceeds() {
	s.processorA.
		On("Commit").
		Return(nil).Once()
	s.processorB.
		On("Commit").
		Return(nil).Once()

	err := s.processors.Commit()
	s.Assert().NoError(err)
}
