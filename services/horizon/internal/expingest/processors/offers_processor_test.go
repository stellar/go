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

func TestOffersrocessorTestSuiteState(t *testing.T) {
	suite.Run(t, new(OffersProcessorTestSuiteState))
}

type OffersProcessorTestSuiteState struct {
	suite.Suite
	processor              *DatabaseProcessor
	mockQ                  *history.MockQOffers
	mockBatchInsertBuilder *history.MockOffersBatchInsertBuilder
	mockStateReader        *io.MockStateReader
	mockStateWriter        *io.MockStateWriter
}

func (s *OffersProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQOffers{}
	s.mockBatchInsertBuilder = &history.MockOffersBatchInsertBuilder{}
	s.mockStateReader = &io.MockStateReader{}
	s.mockStateWriter = &io.MockStateWriter{}

	s.processor = &DatabaseProcessor{
		Action:  Offers,
		OffersQ: s.mockQ,
	}

	// Reader and Writer should be always closed and once
	s.mockStateReader.On("Close").Return(nil).Once()
	s.mockStateWriter.On("Close").Return(nil).Once()

	s.mockQ.
		On("NewOffersBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()
}

func (s *OffersProcessorTestSuiteState) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
	s.mockStateReader.AssertExpectations(s.T())
	s.mockStateWriter.AssertExpectations(s.T())
}

func (s *OffersProcessorTestSuiteState) TestCreateOffer() {
	offer := xdr.OfferEntry{
		OfferId: xdr.Int64(1),
		Price:   xdr.Price{1, 2},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	s.mockStateReader.
		On("Read").Return(
		xdr.LedgerEntryChange{
			Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
			State: &xdr.LedgerEntry{
				Data: xdr.LedgerEntryData{
					Type:  xdr.LedgerEntryTypeOffer,
					Offer: &offer,
				},
				LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			},
		},
		nil,
	).Once()

	s.mockBatchInsertBuilder.
		On("Add", offer, lastModifiedLedgerSeq).Return(nil).Once()

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

func TestOffersProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(OffersProcessorTestSuiteLedger))
}

type OffersProcessorTestSuiteLedger struct {
	suite.Suite
	processor        *DatabaseProcessor
	mockQ            *history.MockQOffers
	mockLedgerReader *io.MockLedgerReader
	mockLedgerWriter *io.MockLedgerWriter
}

func (s *OffersProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQOffers{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}

	s.processor = &DatabaseProcessor{
		Action:  Offers,
		OffersQ: s.mockQ,
	}

	// Reader and Writer should be always closed and once
	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()
}

func (s *OffersProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *OffersProcessorTestSuiteLedger) TestInsertOffer() {
	// should be ignored because it's not an offer type
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

	// should be ignored because transaction was not successful
	s.mockLedgerReader.On("Read").
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
							Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
							Created: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeOffer,
									Offer: &xdr.OfferEntry{
										OfferId: xdr.Int64(6),
										Price:   xdr.Price{1, 2},
									},
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	// add offer
	offer := xdr.OfferEntry{
		OfferId: xdr.Int64(2),
		Price:   xdr.Price{1, 2},
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
								Data: xdr.LedgerEntryData{
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &offer,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockLedgerReader.On("GetSequence").Return(uint32(lastModifiedLedgerSeq))

	s.mockQ.On(
		"InsertOffer",
		offer,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()

	updatedOffer := xdr.OfferEntry{
		OfferId: xdr.Int64(2),
		Price:   xdr.Price{1, 6},
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
								Data: xdr.LedgerEntryData{
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &offer,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &updatedOffer,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockQ.On(
		"UpdateOffer",
		updatedOffer,
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

func (s *OffersProcessorTestSuiteLedger) TestUpdateOfferNoRowsAffected() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)
	s.mockLedgerReader.On("GetSequence").Return(uint32(lastModifiedLedgerSeq))

	offer := xdr.OfferEntry{
		OfferId: xdr.Int64(2),
		Price:   xdr.Price{1, 2},
	}
	updatedOffer := xdr.OfferEntry{
		OfferId: xdr.Int64(2),
		Price:   xdr.Price{1, 6},
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
								Data: xdr.LedgerEntryData{
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &offer,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &updatedOffer,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockQ.On(
		"UpdateOffer",
		updatedOffer,
		lastModifiedLedgerSeq,
	).Return(int64(0), nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(verify.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "Error in processLedgerOffers: No rows affected when updating offer 2")
}

func (s *OffersProcessorTestSuiteLedger) TestRemoveOffer() {
	// add offer
	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeOffer,
									Offer: &xdr.OfferEntry{
										OfferId: xdr.Int64(3),
										Price:   xdr.Price{3, 1},
									},
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeOffer,
								Offer: &xdr.LedgerKeyOffer{
									OfferId: xdr.Int64(3),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockLedgerReader.On("GetSequence").Return(uint32(123))

	s.mockQ.On(
		"RemoveOffer",
		xdr.Int64(3),
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

func (s *OffersProcessorTestSuiteLedger) TestRemoveOfferNoRowsAffected() {
	// add offer
	s.mockLedgerReader.On("Read").
		Return(io.LedgerTransaction{
			Meta: createTransactionMeta([]xdr.OperationMeta{
				xdr.OperationMeta{
					Changes: []xdr.LedgerEntryChange{
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
							State: &xdr.LedgerEntry{
								Data: xdr.LedgerEntryData{
									Type: xdr.LedgerEntryTypeOffer,
									Offer: &xdr.OfferEntry{
										OfferId: xdr.Int64(3),
										Price:   xdr.Price{3, 1},
									},
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeOffer,
								Offer: &xdr.LedgerKeyOffer{
									OfferId: xdr.Int64(3),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockLedgerReader.On("GetSequence").Return(uint32(123))

	s.mockQ.On(
		"RemoveOffer",
		xdr.Int64(3),
	).Return(int64(0), nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(verify.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "Error in processLedgerOffers: No rows affected when removing offer 3")
}
