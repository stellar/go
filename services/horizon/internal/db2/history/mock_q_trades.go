package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockQTrades struct {
	mock.Mock
}

func (m *MockQTrades) CheckExpTrades(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}

func (m *MockQTrades) CreateExpAccounts(addresses []string) (map[string]int64, error) {
	a := m.Called(addresses)
	return a.Get(0).(map[string]int64), a.Error(1)
}

func (m *MockQTrades) CreateExpAssets(assets []xdr.Asset) ([]Asset, error) {
	a := m.Called(assets)
	return a.Get(0).([]Asset), a.Error(1)
}

func (m *MockQTrades) NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(TradeBatchInsertBuilder)
}

type MockTradeBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTradeBatchInsertBuilder) Add(entries ...InsertTrade) error {
	a := m.Called(entries)
	return a.Error(0)
}

func (m *MockTradeBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
