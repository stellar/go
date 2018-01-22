package queue

import (
	"github.com/stretchr/testify/mock"
)

// MockQueue is a mockable queue.
type MockQueue struct {
	mock.Mock
}

func (m *MockQueue) QueueAdd(tx Transaction) error {
	a := m.Called(tx)
	return a.Error(0)
}

func (m *MockQueue) WithQueuedTransaction(f func(Transaction) error) error {
	a := m.Called()
	if err := a.Error(1); err != nil {
		return err
	}

	return f(a.Get(0).(Transaction))
}

func (m *MockQueue) IsEmpty() (bool, error) {
	a := m.Called()
	if err := a.Error(1); err != nil {
		return false, err
	}

	return a.Get(0).(bool), nil
}
