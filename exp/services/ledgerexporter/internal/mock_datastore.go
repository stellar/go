package exporter

import (
	"io"

	"github.com/stretchr/testify/mock"
)

// MockDataStore is a mock implementation for the Storage interface.
type MockDataStore struct {
	mock.Mock
}

func (m *MockDataStore) Exists(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func (m *MockDataStore) Size(path string) (int64, error) {
	args := m.Called(path)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDataStore) GetFile(path string) (io.ReadCloser, error) {
	args := m.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockDataStore) PutFile(path string, in io.ReadCloser) error {
	args := m.Called(path, in)
	return args.Error(0)
}

func (m *MockDataStore) PutFileIfNotExists(path string, in io.ReadCloser) error {
	args := m.Called(path, in)
	return args.Error(0)
}

func (m *MockDataStore) Close() error {
	args := m.Called()
	return args.Error(0)
}
