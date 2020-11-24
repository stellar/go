package history

import (
	"github.com/stretchr/testify/mock"
)

// MockQOffers is a mock implementation of the QOffers interface
type MockQOffers struct {
	mock.Mock
}

func (m *MockQOffers) GetAllOffers() ([]Offer, error) {
	a := m.Called()
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) GetOffersByIDs(ids []int64) ([]Offer, error) {
	a := m.Called(ids)
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) GetUpdatedOffers(newerThanSequence uint32) ([]Offer, error) {
	a := m.Called(newerThanSequence)
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) CountOffers() (int, error) {
	a := m.Called()
	return a.Get(0).(int), a.Error(1)
}

func (m *MockQOffers) NewOffersBatchInsertBuilder(maxBatchSize int) OffersBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OffersBatchInsertBuilder)
}

func (m *MockQOffers) UpdateOffer(row Offer) (int64, error) {
	a := m.Called(row)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQOffers) RemoveOffers(offerIDs []int64, lastModifiedLedger uint32) (int64, error) {
	a := m.Called(offerIDs, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQOffers) CompactOffers(cutOffSequence uint32) (int64, error) {
	a := m.Called(cutOffSequence)
	return a.Get(0).(int64), a.Error(1)
}
