package processors

import (
	"testing"

	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
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
	processor              *OffersProcessor
	mockQ                  *history.MockQOffers
	mockBatchInsertBuilder *history.MockOffersBatchInsertBuilder
	sequence               uint32
}

func (s *OffersProcessorTestSuiteState) SetupTest() {
	s.mockQ = &history.MockQOffers{}
	s.mockBatchInsertBuilder = &history.MockOffersBatchInsertBuilder{}

	s.mockQ.
		On("NewOffersBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.sequence = 456
	s.processor = NewOffersProcessor(s.mockQ, s.sequence)
}

func (s *OffersProcessorTestSuiteState) TearDownTest() {
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockQ.On("CompactOffers", s.sequence-100).Return(int64(0), nil).Once()
	s.Assert().NoError(s.processor.Commit())

	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *OffersProcessorTestSuiteState) TestCreateOffer() {
	offer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(1),
		Price:    xdr.Price{1, 2},
	}
	lastModifiedLedgerSeq := xdr.Uint32(123)
	s.mockBatchInsertBuilder.
		On("Add", offer, lastModifiedLedgerSeq).Return(nil).Once()

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:  xdr.LedgerEntryTypeOffer,
				Offer: &offer,
			},
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
		},
	})
	s.Assert().NoError(err)
}

func TestOffersProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(OffersProcessorTestSuiteLedger))
}

type OffersProcessorTestSuiteLedger struct {
	suite.Suite
	processor              *OffersProcessor
	mockQ                  *history.MockQOffers
	mockBatchInsertBuilder *history.MockOffersBatchInsertBuilder
	sequence               uint32
}

func (s *OffersProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQOffers{}
	s.mockBatchInsertBuilder = &history.MockOffersBatchInsertBuilder{}

	s.mockQ.
		On("NewOffersBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	s.sequence = 456
	s.processor = NewOffersProcessor(s.mockQ, s.sequence)
}

func (s *OffersProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *OffersProcessorTestSuiteLedger) setupInsertOffer() {
	// should be ignored because it's not an offer type
	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:  xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
	})
	s.Assert().NoError(err)

	// add offer
	offer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 2},
	}
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	err = s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:  xdr.LedgerEntryTypeOffer,
				Offer: &offer,
			},
		},
	})
	s.Assert().NoError(err)

	updatedOffer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 6},
	}

	err = s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
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
	})
	s.Assert().NoError(err)

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockBatchInsertBuilder.On(
		"Add",
		updatedOffer,
		lastModifiedLedgerSeq,
	).Return(nil).Once()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
}

func (s *OffersProcessorTestSuiteLedger) TestInsertOffer() {
	s.setupInsertOffer()
	s.mockQ.On("CompactOffers", s.sequence-100).Return(int64(0), nil).Once()
	s.Assert().NoError(s.processor.Commit())
}

func (s *OffersProcessorTestSuiteLedger) TestSkipCompactionIfSequenceEqualsWindow() {
	s.processor.sequence = offerCompactionWindow
	s.setupInsertOffer()
	s.Assert().NoError(s.processor.Commit())
}

func (s *OffersProcessorTestSuiteLedger) TestSkipCompactionIfSequenceLessThanWindow() {
	s.processor.sequence = offerCompactionWindow - 1
	s.setupInsertOffer()
	s.Assert().NoError(s.processor.Commit())
}

func (s *OffersProcessorTestSuiteLedger) TestCompactionError() {
	s.setupInsertOffer()
	s.mockQ.On("CompactOffers", s.sequence-100).
		Return(int64(0), errors.New("compaction error")).Once()
	s.Assert().EqualError(s.processor.Commit(), "could not compact offers: compaction error")
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

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq - 1,
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
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"UpdateOffer",
		updatedOffer,
		lastModifiedLedgerSeq,
	).Return(int64(0), nil).Once()

	err = s.processor.Commit()
	s.Assert().Error(err)
	s.Assert().IsType(ingesterrors.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "error flushing cache: 0 rows affected when updating offer 2")
}

func (s *OffersProcessorTestSuiteLedger) TestRemoveOffer() {
	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					OfferId:  xdr.Int64(3),
					Price:    xdr.Price{3, 1},
				},
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"RemoveOffer",
		xdr.Int64(3),
		s.sequence,
	).Return(int64(1), nil).Once()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockQ.On("CompactOffers", s.sequence-100).Return(int64(0), nil).Once()
	s.Assert().NoError(s.processor.Commit())
}

func (s *OffersProcessorTestSuiteLedger) TestProcessUpgradeChange() {
	// add offer
	offer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 2},
	}
	lastModifiedLedgerSeq := xdr.Uint32(1234)

	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: lastModifiedLedgerSeq,
			Data: xdr.LedgerEntryData{
				Type:  xdr.LedgerEntryTypeOffer,
				Offer: &offer,
			},
		},
	})
	s.Assert().NoError(err)

	updatedOffer := xdr.OfferEntry{
		SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
		OfferId:  xdr.Int64(2),
		Price:    xdr.Price{1, 6},
	}

	err = s.processor.ProcessChange(io.Change{
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
	})
	s.Assert().NoError(err)

	// We use LedgerEntryChangesCache so all changes are squashed
	s.mockBatchInsertBuilder.On(
		"Add",
		updatedOffer,
		lastModifiedLedgerSeq,
	).Return(nil).Once()

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()
	s.mockQ.On("CompactOffers", s.sequence-100).Return(int64(0), nil).Once()
	s.Assert().NoError(s.processor.Commit())
}

func (s *OffersProcessorTestSuiteLedger) TestRemoveOfferNoRowsAffected() {
	err := s.processor.ProcessChange(io.Change{
		Type: xdr.LedgerEntryTypeOffer,
		Pre: &xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					SellerId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					OfferId:  xdr.Int64(3),
					Price:    xdr.Price{3, 1},
				},
			},
		},
		Post: nil,
	})
	s.Assert().NoError(err)

	s.mockQ.On(
		"RemoveOffer",
		xdr.Int64(3),
		s.sequence,
	).Return(int64(0), nil).Once()

	err = s.processor.Commit()
	s.Assert().Error(err)
	s.Assert().IsType(ingesterrors.StateError{}, errors.Cause(err))
	s.Assert().EqualError(err, "error flushing cache: 0 rows affected when removing offer 3")
}
