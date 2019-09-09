package history

import (
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQOffers is a mock implementation of the QOffers interface
type MockQOffers struct {
	mock.Mock
}

func (m *MockQOffers) GetAllOffers() ([]Offer, error) {
	a := m.Called()
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) NewOffersBatchInsertBuilder(maxBatchSize int) OffersBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OffersBatchInsertBuilder)
}

func (m *MockQOffers) UpsertOffer(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) error {
	a := m.Called(offer, lastModifiedLedger)
	return a.Error(0)
}

func (m *MockQOffers) RemoveOffer(offerID xdr.Int64) error {
	a := m.Called(offerID)
	return a.Error(0)
}
