package io

import (
	"github.com/stretchr/testify/mock"
)

var _ ChangeReader = (*MockChangeReader)(nil)

type MockChangeReader struct {
	mock.Mock
}

func (m *MockChangeReader) Read() (Change, error) {
	args := m.Called()
	return args.Get(0).(Change), args.Error(1)
}
