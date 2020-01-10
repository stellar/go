package processors

import (
	"context"
	stdio "io"
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ParticipantsProcessorTestSuiteLedger struct {
	suite.Suite
	processor                        *ParticipantsProcessor
	mockQ                            *history.MockQParticipants
	mockBatchInsertBuilder           *history.MockTransactionParticipantsBatchInsertBuilder
	mockOperationsBatchInsertBuilder *history.MockOperationParticipantBatchInsertBuilder
	mockLedgerReader                 *io.MockLedgerReader
	mockLedgerWriter                 *io.MockLedgerWriter
	context                          context.Context

	firstTx     io.LedgerTransaction
	secondTx    io.LedgerTransaction
	thirdTx     io.LedgerTransaction
	firstTxID   int64
	secondTxID  int64
	thirdTxID   int64
	sequence    uint32
	addresses   []string
	addressToID map[string]int64
}

func TestParticipantsProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(ParticipantsProcessorTestSuiteLedger))
}

func (s *ParticipantsProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQParticipants{}
	s.mockBatchInsertBuilder = &history.MockTransactionParticipantsBatchInsertBuilder{}
	s.mockOperationsBatchInsertBuilder = &history.MockOperationParticipantBatchInsertBuilder{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}
	s.context = context.WithValue(context.Background(), IngestUpdateDatabase, true)

	s.sequence = uint32(20)

	s.addresses = []string{
		"GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
		"GAXI33UCLQTCKM2NMRBS7XYBR535LLEVAHL5YBN4FTCB4HZHT7ZA5CVK",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	}

	s.firstTx = createTransaction(true, 1)
	s.firstTx.Index = 1
	s.firstTx.Envelope.Tx.SourceAccount = xdr.MustAddress(s.addresses[0])
	s.firstTxID = toid.New(int32(s.sequence), 1, 0).ToInt64()

	s.secondTx = createTransaction(true, 1)
	s.secondTx.Index = 2
	s.secondTx.Envelope.Tx.Operations[0].Body = xdr.OperationBody{
		Type: xdr.OperationTypeCreateAccount,
		CreateAccountOp: &xdr.CreateAccountOp{
			Destination: xdr.MustAddress(s.addresses[1]),
		},
	}
	s.secondTx.Envelope.Tx.SourceAccount = xdr.MustAddress(s.addresses[2])
	s.secondTxID = toid.New(int32(s.sequence), 2, 0).ToInt64()

	s.thirdTx = createTransaction(true, 1)
	s.thirdTx.Index = 3
	s.thirdTx.Envelope.Tx.SourceAccount = xdr.MustAddress(s.addresses[0])
	s.thirdTxID = toid.New(int32(s.sequence), 3, 0).ToInt64()

	s.addressToID = map[string]int64{
		s.addresses[0]: 2,
		s.addresses[1]: 20,
		s.addresses[2]: 200,
	}

	s.processor = &ParticipantsProcessor{
		ParticipantsQ: s.mockQ,
	}

	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()
	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()
}

func (s *ParticipantsProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
	s.mockOperationsBatchInsertBuilder.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *ParticipantsProcessorTestSuiteLedger) mockLedgerReads() {
	s.mockLedgerReader.
		On("Read").
		Return(s.firstTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.secondTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(s.thirdTx, nil).Once()
	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()
}

func (s *ParticipantsProcessorTestSuiteLedger) mockSuccessfulTransactionBatchAdds() {
	s.mockBatchInsertBuilder.On(
		"Add", s.firstTxID, s.addressToID[s.addresses[0]],
	).Return(nil).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[1]],
	).Return(nil).Once()
	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[2]],
	).Return(nil).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.thirdTxID, s.addressToID[s.addresses[0]],
	).Return(nil).Once()
}

func (s *ParticipantsProcessorTestSuiteLedger) mockSuccessfulOperationBatchAdds() {
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.firstTxID+1, s.addressToID[s.addresses[0]],
	).Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[1]],
	).Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[2]],
	).Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.thirdTxID+1, s.addressToID[s.addresses[0]],
	).Return(nil).Once()
}
func (s *ParticipantsProcessorTestSuiteLedger) TestNoIngestUpdateDatabase() {
	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *ParticipantsProcessorTestSuiteLedger) TestEmptyParticipants() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockQ.On("CheckExpParticipants", int32(s.sequence-10)).
		Return(true, nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *ParticipantsProcessorTestSuiteLedger) TestCheckExpParticipantsError() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockQ.On("CheckExpParticipants", int32(s.sequence-10)).
		Return(false, errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *ParticipantsProcessorTestSuiteLedger) TestParticipantsCheckDoesNotMatch() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockQ.On("CheckExpParticipants", int32(s.sequence-10)).
		Return(false, nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *ParticipantsProcessorTestSuiteLedger) TestIngestParticipantsSucceeds() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReads()

	s.mockQ.On("CreateExpAccounts", mock.AnythingOfType("[]string")).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.addresses,
				arg,
			)
		}).Return(s.addressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
	s.mockQ.On("NewOperationParticipantBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationsBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()
	s.mockSuccessfulOperationBatchAdds()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On("Exec").Return(nil).Once()

	s.mockQ.On("CheckExpParticipants", int32(s.sequence-10)).
		Return(true, nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().NoError(err)
}

func (s *ParticipantsProcessorTestSuiteLedger) TestCreateExpAccountsFails() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReads()

	s.mockQ.On("CreateExpAccounts", mock.AnythingOfType("[]string")).
		Return(s.addressToID, errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().EqualError(err, "Could not create account ids: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestBatchAddFails() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReads()

	s.mockQ.On("CreateExpAccounts", mock.AnythingOfType("[]string")).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.addresses,
				arg,
			)
		}).Return(s.addressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.firstTxID, s.addressToID[s.addresses[0]],
	).Return(errors.New("transient error")).Once()

	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[1]],
	).Return(nil).Maybe()
	s.mockBatchInsertBuilder.On(
		"Add", s.secondTxID, s.addressToID[s.addresses[2]],
	).Return(nil).Maybe()

	s.mockBatchInsertBuilder.On(
		"Add", s.thirdTxID, s.addressToID[s.addresses[0]],
	).Return(nil).Maybe()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().EqualError(err, "Could not insert transaction participant in db: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestOperationParticipantsBatchAddFails() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReads()

	s.mockQ.On("CreateExpAccounts", mock.AnythingOfType("[]string")).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.addresses,
				arg,
			)
		}).Return(s.addressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
	s.mockQ.On("NewOperationParticipantBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationsBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()

	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.firstTxID+1, s.addressToID[s.addresses[0]],
	).Return(errors.New("transient error")).Once()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[1]],
	).Return(nil).Maybe()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.secondTxID+1, s.addressToID[s.addresses[2]],
	).Return(nil).Maybe()
	s.mockOperationsBatchInsertBuilder.On(
		"Add", s.thirdTxID+1, s.addressToID[s.addresses[0]],
	).Return(nil).Maybe()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().EqualError(err, "could not insert operation participant in db: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestBatchAddExecFails() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReads()

	s.mockQ.On("CreateExpAccounts", mock.AnythingOfType("[]string")).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.addresses,
				arg,
			)
		}).Return(s.addressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()

	s.mockBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().EqualError(err, "Could not flush transaction participants to db: transient error")
}

func (s *ParticipantsProcessorTestSuiteLedger) TestOpeartionBatchAddExecFails() {
	s.mockLedgerReader.On("GetSequence").Return(s.sequence).Once()

	s.mockLedgerReads()

	s.mockQ.On("CreateExpAccounts", mock.AnythingOfType("[]string")).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				s.addresses,
				arg,
			)
		}).Return(s.addressToID, nil).Once()
	s.mockQ.On("NewTransactionParticipantsBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
	s.mockQ.On("NewOperationParticipantBatchInsertBuilder", maxBatchSize).
		Return(s.mockOperationsBatchInsertBuilder).Once()

	s.mockSuccessfulTransactionBatchAdds()
	s.mockSuccessfulOperationBatchAdds()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockOperationsBatchInsertBuilder.On("Exec").Return(errors.New("transient error")).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)
	s.Assert().EqualError(err, "could not flush operation participants to db: transient error")
}
