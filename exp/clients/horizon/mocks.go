package horizonclient

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockClient is a mockable horizon client.
type MockClient struct {
	mock.Mock
}

// AccountDetail is a mocking method
func (m *MockClient) AccountDetail(request AccountRequest) (Account, error) {
	a := m.Called(request)
	return a.Get(0).(Account), a.Error(1)
}

// AccountData is a mocking method
func (m *MockClient) AccountData(request AccountRequest) (AccountData, error) {
	a := m.Called(request)
	return a.Get(0).(AccountData), a.Error(1)
}

// Effects is a mocking method
func (m *MockClient) Effects(request EffectRequest) (EffectsPage, error) {
	a := m.Called(request)
	return a.Get(0).(EffectsPage), a.Error(1)
}

// Assets is a mocking method
func (m *MockClient) Assets(request AssetRequest) (AssetsPage, error) {
	a := m.Called(request)
	return a.Get(0).(AssetsPage), a.Error(1)
}

func (m *MockClient) Stream(ctx context.Context,
	request StreamRequest,
	handler func(interface{}),
) error {
	a := m.Called(request, ctx, handler)
	return a.Error(0)
}

func (m *MockClient) Ledgers(request LedgerRequest) (LedgersPage, error) {
	a := m.Called(request)
	return a.Get(0).(LedgersPage), a.Error(1)
}

func (m *MockClient) LedgerDetail(sequence uint32) (Ledger, error) {
	a := m.Called(sequence)
	return a.Get(0).(Ledger), a.Error(1)
}

func (m *MockClient) Metrics() (Metrics, error) {
	a := m.Called()
	return a.Get(0).(Metrics), a.Error(1)
}

// ensure that the MockClient implements ClientInterface
var _ ClientInterface = &MockClient{}
