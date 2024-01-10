package storage

import (
	"github.com/stretchr/testify/mock"
	"io"
)

// MockStorage is a mock implementation for the Storage interface.
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Exists(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorage) Size(path string) (int64, error) {
	args := m.Called(path)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStorage) GetFile(path string) (io.ReadCloser, error) {
	args := m.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) PutFile(path string, in io.ReadCloser) error {
	args := m.Called(path, in)
	return args.Error(0)
}

func (m *MockStorage) ListFiles(path string) (chan string, chan error) {
	args := m.Called(path)
	return args.Get(0).(chan string), args.Get(1).(chan error)
}

func (m *MockStorage) CanListFiles() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}
