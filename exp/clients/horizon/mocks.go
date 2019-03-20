package horizonclient

import (
	"context"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mockable horizon client.
type MockClient struct {
	mock.Mock
}

// AccountDetail is a mocking method
func (m *MockClient) AccountDetail(request AccountRequest) (hProtocol.Account, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.Account), a.Error(1)
}

// AccountData is a mocking method
func (m *MockClient) AccountData(request AccountRequest) (hProtocol.AccountData, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.AccountData), a.Error(1)
}

// Effects is a mocking method
func (m *MockClient) Effects(request EffectRequest) (hProtocol.EffectsPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.EffectsPage), a.Error(1)
}

// Assets is a mocking method
func (m *MockClient) Assets(request AssetRequest) (hProtocol.AssetsPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.AssetsPage), a.Error(1)
}

func (m *MockClient) Stream(ctx context.Context,
	request StreamRequest,
	handler func(interface{}),
) error {
	a := m.Called(request, ctx, handler)
	return a.Error(0)
}

func (m *MockClient) Ledgers(request LedgerRequest) (hProtocol.LedgersPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.LedgersPage), a.Error(1)
}

func (m *MockClient) LedgerDetail(sequence uint32) (hProtocol.Ledger, error) {
	a := m.Called(sequence)
	return a.Get(0).(hProtocol.Ledger), a.Error(1)
}

func (m *MockClient) Metrics() (hProtocol.Metrics, error) {
	a := m.Called()
	return a.Get(0).(hProtocol.Metrics), a.Error(1)
}

func (m *MockClient) FeeStats() (hProtocol.FeeStats, error) {
	a := m.Called()
	return a.Get(0).(hProtocol.FeeStats), a.Error(1)
}

func (m *MockClient) Offers(request OfferRequest) (hProtocol.OffersPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.OffersPage), a.Error(1)
}

func (m *MockClient) Operations(request OperationRequest) (operations.OperationsPage, error) {
	a := m.Called(request)
	return a.Get(0).(operations.OperationsPage), a.Error(1)
}

func (m *MockClient) OperationDetail(id string) (operations.Operation, error) {
	a := m.Called(id)
	return a.Get(0).(operations.Operation), a.Error(1)
}

// ensure that the MockClient implements ClientInterface
var _ ClientInterface = &MockClient{}
