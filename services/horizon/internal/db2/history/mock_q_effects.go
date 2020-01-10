package history

import (
	"github.com/stretchr/testify/mock"
)

// MockQEffects is a mock implementation of the QEffects interface
type MockQEffects struct {
	mock.Mock
}

func (m *MockQEffects) NewEffectBatchInsertBuilder(maxBatchSize int) EffectBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(EffectBatchInsertBuilder)
}

func (m *MockQEffects) CreateExpAccounts(addresses []string) (map[string]int64, error) {
	a := m.Called(addresses)
	return a.Get(0).(map[string]int64), a.Error(1)
}

func (m *MockQEffects) CheckExpOperationEffects(seq int32) (bool, error) {
	a := m.Called(seq)
	return a.Get(0).(bool), a.Error(1)
}
