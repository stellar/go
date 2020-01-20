package processors

import (
	"context"
	stdio "io"
	"testing"

	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	supportPipeline "github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestOffersProcessorTestSuiteState(t *testing.T) {
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
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(1),
		Price:    xdr.Price{1, 2},
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
	context          context.Context
	processor        *DatabaseProcessor
	mockQ            *history.MockQOffers
	mockLedgerReader *io.MockLedgerReader
	mockLedgerWriter *io.MockLedgerWriter
}

func (s *OffersProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQOffers{}
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}

	s.context = context.WithValue(context.Background(), IngestUpdateState, true)

	s.processor = &DatabaseProcessor{
		Action:  Offers,
		OffersQ: s.mockQ,
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

func (s *OffersProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockLedgerReader.AssertExpectations(s.T())
	s.mockLedgerWriter.AssertExpectations(s.T())
}

func (s *OffersProcessorTestSuiteLedger) TestNoIngestUpdateState() {
	s.mockLedgerReader = &io.MockLedgerReader{}
	s.mockLedgerWriter = &io.MockLedgerWriter{}

	s.mockLedgerReader.On("IgnoreUpgradeChanges").Once()

	s.mockLedgerReader.
		On("Close").
		Return(nil).Once()

	s.mockLedgerWriter.
		On("Close").
		Return(nil).Once()

	err := s.processor.ProcessLedger(
		context.Background(),
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *OffersProcessorTestSuiteLedger) TestInsertOffer() {
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
	// should be ignored because it's not an offer type
	changes, err := accountTransaction.GetChanges()
	s.Assert().NoError(err)
	s.Assert().NoError(s.processor.processLedgerOffers(changes[0]))

	// add offer
	offer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 2},
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
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &offer,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	updatedOffer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 6},
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
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &offer,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
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

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockQ.On(
		"InsertOffer",
		updatedOffer,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	err = s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *OffersProcessorTestSuiteLedger) TestUpdateOfferNoRowsAffected() {
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	offer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 2},
	}
	updatedOffer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 6},
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
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &offer,
								},
							},
						},
						// Updated
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
							Updated: &xdr.LedgerEntry{
								LastModifiedLedgerSeq: lastModifiedLedgerSeq,
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
	s.mockLedgerReader.On("Read").Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockQ.On(
		"UpdateOffer",
		updatedOffer,
		lastModifiedLedgerSeq,
	).Return(int64(0), nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(ingesterrors.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "Error in Offers handler: 0 rows affected when updating offer 2")
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
										SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										OfferId:  xdr.Int64(3),
										Price:    xdr.Price{3, 1},
									},
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeOffer,
								Offer: &xdr.LedgerKeyOffer{
									SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
									OfferId:  xdr.Int64(3),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockQ.On(
		"RemoveOffer",
		xdr.Int64(3),
	).Return(int64(1), nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().NoError(err)
}

func (s *OffersProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	// Removes ReadUpgradeChange assertion
	s.mockLedgerReader = &io.MockLedgerReader{}

	// add offer
	offer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 2},
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
									Type:  xdr.LedgerEntryTypeOffer,
									Offer: &offer,
								},
							},
						},
					},
				},
			}),
		}, nil).Once()

	s.mockLedgerReader.
		On("Read").
		Return(io.LedgerTransaction{}, stdio.EOF).Once()

	updatedOffer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 6},
	}

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(
			io.Change{
				Type: xdr.LedgerEntryTypeOffer,
				Pre: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: lastModifiedLedgerSeq,
					Data: xdr.LedgerEntryData{
						Type:  xdr.LedgerEntryTypeOffer,
						Offer: &offer,
					},
				},
				Post: &xdr.LedgerEntry{
					LastModifiedLedgerSeq: lastModifiedLedgerSeq,
					Data: xdr.LedgerEntryData{
						Type:  xdr.LedgerEntryTypeOffer,
						Offer: &updatedOffer,
					},
				},
			}, nil).Once()

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockQ.On(
		"InsertOffer",
		updatedOffer,
		lastModifiedLedgerSeq,
	).Return(int64(1), nil).Once()

	s.mockLedgerReader.
		On("ReadUpgradeChange").
		Return(io.Change{}, stdio.EOF).Once()

	s.mockLedgerReader.On("Close").Return(nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
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
										SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
										OfferId:  xdr.Int64(3),
										Price:    xdr.Price{3, 1},
									},
								},
							},
						},
						xdr.LedgerEntryChange{
							Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
							Removed: &xdr.LedgerKey{
								Type: xdr.LedgerEntryTypeOffer,
								Offer: &xdr.LedgerKeyOffer{
									SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
									OfferId:  xdr.Int64(3),
								},
							},
						},
					},
				},
			}),
		}, nil).Once()
	s.mockLedgerReader.On("Read").Return(io.LedgerTransaction{}, stdio.EOF).Once()

	s.mockQ.On(
		"RemoveOffer",
		xdr.Int64(3),
	).Return(int64(0), nil).Once()

	err := s.processor.ProcessLedger(
		s.context,
		&supportPipeline.Store{},
		s.mockLedgerReader,
		s.mockLedgerWriter,
	)

	s.Assert().Error(err)
	s.Assert().IsType(ingesterrors.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "Error in Offers handler: 0 rows affected when removing offer 3")
}
