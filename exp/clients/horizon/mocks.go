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

// Stream is a mocking method
func (m *MockClient) Stream(ctx context.Context,
	request StreamRequest,
	handler func(interface{}),
) error {
	a := m.Called(ctx, request, handler)
	return a.Error(0)
}

// Ledgers is a mocking method
func (m *MockClient) Ledgers(request LedgerRequest) (hProtocol.LedgersPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.LedgersPage), a.Error(1)
}

// LedgerDetail is a mocking method
func (m *MockClient) LedgerDetail(sequence uint32) (hProtocol.Ledger, error) {
	a := m.Called(sequence)
	return a.Get(0).(hProtocol.Ledger), a.Error(1)
}

// Metrics is a mocking method
func (m *MockClient) Metrics() (hProtocol.Metrics, error) {
	a := m.Called()
	return a.Get(0).(hProtocol.Metrics), a.Error(1)
}

// FeeStats is a mocking method
func (m *MockClient) FeeStats() (hProtocol.FeeStats, error) {
	a := m.Called()
	return a.Get(0).(hProtocol.FeeStats), a.Error(1)
}

// Offers is a mocking method
func (m *MockClient) Offers(request OfferRequest) (hProtocol.OffersPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.OffersPage), a.Error(1)
}

// Operations is a mocking method
func (m *MockClient) Operations(request OperationRequest) (operations.OperationsPage, error) {
	a := m.Called(request)
	return a.Get(0).(operations.OperationsPage), a.Error(1)
}

// OperationDetail is a mocking method
func (m *MockClient) OperationDetail(id string) (operations.Operation, error) {
	a := m.Called(id)
	return a.Get(0).(operations.Operation), a.Error(1)
}

// SubmitTransaction is a mocking method
func (m *MockClient) SubmitTransaction(transactionXdr string) (hProtocol.TransactionSuccess, error) {
	a := m.Called(transactionXdr)
	return a.Get(0).(hProtocol.TransactionSuccess), a.Error(1)
}

// Transactions is a mocking method
func (m *MockClient) Transactions(request TransactionRequest) (hProtocol.TransactionsPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.TransactionsPage), a.Error(1)
}

// TransactionDetail is a mocking method
func (m *MockClient) TransactionDetail(txHash string) (hProtocol.Transaction, error) {
	a := m.Called(txHash)
	return a.Get(0).(hProtocol.Transaction), a.Error(1)
}

// OrderBook is a mocking method
func (m *MockClient) OrderBook(request OrderBookRequest) (hProtocol.OrderBookSummary, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.OrderBookSummary), a.Error(1)
}

// Paths is a mocking method
func (m *MockClient) Paths(request PathsRequest) (hProtocol.PathsPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.PathsPage), a.Error(1)
}

// Payments is a mocking method
func (m *MockClient) Payments(request OperationRequest) (operations.OperationsPage, error) {
	a := m.Called(request)
	return a.Get(0).(operations.OperationsPage), a.Error(1)
}

// TradeAggregations is a mocking method
func (m *MockClient) TradeAggregations(request TradeAggregationRequest) (hProtocol.TradeAggregationsPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.TradeAggregationsPage), a.Error(1)
}

// Trades is a mocking method
func (m *MockClient) Trades(request TradeRequest) (hProtocol.TradesPage, error) {
	a := m.Called(request)
	return a.Get(0).(hProtocol.TradesPage), a.Error(1)
}

// StreamTransactions is a mocking method
func (m *MockClient) StreamTransactions(ctx context.Context,
	request TransactionRequest,
	handler TransactionHandler,
) error {
	a := m.Called(ctx, request, handler)
	return a.Error(0)
}

// StreamTrades is a mocking method
func (m *MockClient) StreamTrades(ctx context.Context,
	request TradeRequest,
	handler TradeHandler,
) error {
	a := m.Called(ctx, request, handler)
	return a.Error(0)
}

// StreamEffects is a mocking method
func (m *MockClient) StreamEffects(ctx context.Context,
	request EffectRequest,
	handler EffectHandler,
) error {
	return m.Called(ctx, request, handler).Error(0)
}

// StreamOffers is a mocking method
func (m *MockClient) StreamOffers(ctx context.Context,
	request OfferRequest,
	handler OfferHandler,
) error {
	return m.Called(ctx, request, handler).Error(0)
}

// StreamLedgers is a mocking method
func (m *MockClient) StreamLedgers(ctx context.Context,
	request LedgerRequest,
	handler LedgerHandler,
) error {
	return m.Called(ctx, request, handler).Error(0)
}

// ensure that the MockClient implements ClientInterface
var _ ClientInterface = &MockClient{}
