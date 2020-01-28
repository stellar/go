package io

import (
	"github.com/stretchr/testify/mock"
)

var _ StateReader = (*MockStateReader)(nil)

type MockStateReader struct {
	mock.Mock
}

func (m *MockStateReader) GetSequence() uint32 {
	args := m.Called()
	return args.Get(0).(uint32)
}

func (m *MockStateReader) Read() (Change, error) {
	args := m.Called()
	return args.Get(0).(Change), args.Error(1)
}

func (m *MockStateReader) Close() error {
	args := m.Called()
	return args.Error(0)
}
