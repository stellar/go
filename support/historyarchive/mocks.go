package historyarchive

import (
	"github.com/stretchr/testify/mock"
)

// MockArchive is a mockable archive.
type MockArchive struct {
	mock.Mock
}

func (m *MockArchive) GetPathHAS(path string) (HistoryArchiveState, error) {
	a := m.Called(path)
	return a.Get(0).(HistoryArchiveState), a.Error(1)
}

func (m *MockArchive) PutPathHAS(path string, has HistoryArchiveState, opts *CommandOptions) error {
	a := m.Called(path, has, opts)
	return a.Error(0)
}

func (m *MockArchive) BucketExists(bucket Hash) bool {
	a := m.Called(bucket)
	return a.Get(0).(bool)
}

func (m *MockArchive) CategoryCheckpointExists(cat string, chk uint32) bool {
	a := m.Called(cat, chk)
	return a.Get(0).(bool)
}

func (m *MockArchive) GetRootHAS() (HistoryArchiveState, error) {
	a := m.Called()
	return a.Get(0).(HistoryArchiveState), a.Error(1)
}

func (m *MockArchive) GetCheckpointHAS(chk uint32) (HistoryArchiveState, error) {
	a := m.Called(chk)
	return a.Get(0).(HistoryArchiveState), a.Error(1)
}

func (m *MockArchive) PutCheckpointHAS(chk uint32, has HistoryArchiveState, opts *CommandOptions) error {
	a := m.Called(chk, has, opts)
	return a.Error(0)
}

func (m *MockArchive) PutRootHAS(has HistoryArchiveState, opts *CommandOptions) error {
	a := m.Called(has, opts)
	return a.Error(0)
}

func (m *MockArchive) ListBucket(dp DirPrefix) (chan string, chan error) {
	m.Called(dp)
	panic("Returning channels not implemented")
	return make(chan string), make(chan error)
}

func (m *MockArchive) ListAllBuckets() (chan string, chan error) {
	m.Called()
	panic("Returning channels not implemented")
	return make(chan string), make(chan error)
}

func (m *MockArchive) ListAllBucketHashes() (chan Hash, chan error) {
	m.Called()
	panic("Returning channels not implemented")
	return make(chan Hash), make(chan error)
}

func (m *MockArchive) ListCategoryCheckpoints(cat string, pth string) (chan uint32, chan error) {
	m.Called(cat, pth)
	panic("Returning channels not implemented")
	return make(chan uint32), make(chan error)
}

func (m *MockArchive) GetXdrStreamForHash(hash Hash) (*XdrStream, error) {
	a := m.Called(hash)
	return a.Get(0).(*XdrStream), a.Error(1)
}

func (m *MockArchive) GetXdrStream(pth string) (*XdrStream, error) {
	a := m.Called(pth)
	return a.Get(0).(*XdrStream), a.Error(1)
}
