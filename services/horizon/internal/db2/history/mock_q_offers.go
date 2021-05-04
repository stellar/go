package history

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// MockQOffers is a mock implementation of the QOffers interface
type MockQOffers struct {
	mock.Mock
}

func (m *MockQOffers) GetAllOffers(ctx context.Context) ([]Offer, error) {
	a := m.Called(ctx)
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) GetOffersByIDs(ctx context.Context, ids []int64) ([]Offer, error) {
	a := m.Called(ctx, ids)
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) GetUpdatedOffers(ctx context.Context, newerThanSequence uint32) ([]Offer, error) {
	a := m.Called(ctx, newerThanSequence)
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) CountOffers(ctx context.Context) (int, error) {
	a := m.Called(ctx)
	return a.Get(0).(int), a.Error(1)
}

func (m *MockQOffers) NewOffersBatchInsertBuilder(maxBatchSize int) OffersBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OffersBatchInsertBuilder)
}

func (m *MockQOffers) UpdateOffer(ctx context.Context, row Offer) (int64, error) {
	a := m.Called(ctx, row)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQOffers) RemoveOffers(ctx context.Context, offerIDs []int64, lastModifiedLedger uint32) (int64, error) {
	a := m.Called(ctx, offerIDs, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQOffers) CompactOffers(ctx context.Context, cutOffSequence uint32) (int64, error) {
	a := m.Called(ctx, cutOffSequence)
	return a.Get(0).(int64), a.Error(1)
}
