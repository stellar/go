package ledgerexporter

import (
	"context"
	"io"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

// MockDataStore is a mock implementation for the Storage interface.
type MockDataStore struct {
	mock.Mock
}

func (m *MockDataStore) Exists(ctx context.Context, path string) (bool, error) {
	args := m.Called(ctx, path)
	return args.Bool(0), args.Error(1)
}

func (m *MockDataStore) Size(ctx context.Context, path string) (int64, error) {
	args := m.Called(ctx, path)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDataStore) GetFile(ctx context.Context, path string) (io.ReadCloser, error) {
	args := m.Called(ctx, path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockDataStore) PutFile(ctx context.Context, path string, in io.WriterTo) error {
	args := m.Called(ctx, path, in)
	return args.Error(0)
}

func (m *MockDataStore) PutFileIfNotExists(ctx context.Context, path string, in io.WriterTo) error {
	args := m.Called(ctx, path, in)
	return args.Error(0)
}

func (m *MockDataStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockExportManager struct {
	mock.Mock
}

func (m *MockExportManager) GetMetaArchiveChannel() chan *LedgerMetaArchive {
	a := m.Called()
	return a.Get(0).(chan *LedgerMetaArchive)
}

func (m *MockExportManager) Run(ctx context.Context, startLedger uint32, endLedger uint32) error {
	a := m.Called(ctx, startLedger, endLedger)
	return a.Error(0)
}

func (m *MockExportManager) AddLedgerCloseMeta(ctx context.Context, ledgerCloseMeta xdr.LedgerCloseMeta) error {
	a := m.Called(ctx, ledgerCloseMeta)
	return a.Error(0)
}

type MockNetworkManager struct {
	mock.Mock
}

func (m *MockNetworkManager) GetLatestLedgerSequenceFromHistoryArchives(ctx context.Context, networkName string) (uint32, error) {
	a := m.Called(ctx, networkName)
	return a.Get(0).(uint32), a.Error(1)
}

type MockResumableManager struct {
	mock.Mock
}

func (m *MockResumableManager) FindStartBoundary(ctx context.Context, start, end uint32) uint32 {
	a := m.Called(ctx, start, end)
	return a.Get(0).(uint32)
}

// ensure that the MockClient implements ClientInterface
var _ DataStore = &MockDataStore{}
var _ ExportManager = &MockExportManager{}
var _ NetworkManager = &MockNetworkManager{}
var _ ResumableManager = &MockResumableManager{}
