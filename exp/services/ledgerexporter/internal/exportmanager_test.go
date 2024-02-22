package exporter

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/collections/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestExporterSuite(t *testing.T) {
	suite.Run(t, new(ExportManagerSuite))
}

// ExportManagerSuite is a test suite for the ExportManager.
type ExportManagerSuite struct {
	suite.Suite
	ctx         context.Context
	mockBackend ledgerbackend.MockDatabaseBackend
}

func (s *ExportManagerSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockBackend = ledgerbackend.MockDatabaseBackend{}
}

func (s *ExportManagerSuite) TearDownTest() {
	s.mockBackend.AssertExpectations(s.T())
}

func (s *ExportManagerSuite) TestRun() {
	config := ExporterConfig{LedgersPerFile: 64, FilesPerPartition: 10}
	exporter := NewExportManager(config, &s.mockBackend)

	start := uint32(0)
	end := uint32(255)
	expectedKeys := set.NewSet[string](10)
	for i := start; i <= end; i++ {
		s.mockBackend.On("GetLedger", s.ctx, i).
			Return(createLedgerCloseMeta(i), nil)
		key, _ := GetObjectKeyFromSequenceNumber(config, i)
		expectedKeys.Add(key)
	}

	actualKeys := set.NewSet[string](10)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range exporter.GetMetaArchiveChannel() {
			actualKeys.Add(v.objectKey)
		}
	}()

	err := exporter.Run(s.ctx, start, end)
	assert.NoError(s.T(), err)

	wg.Wait()

	assert.Equal(s.T(), expectedKeys, actualKeys)
}

func (s *ExportManagerSuite) TestRunContextCancel() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 1}
	exporter := NewExportManager(config, &s.mockBackend)
	ctx, cancel := context.WithCancel(context.Background())

	s.mockBackend.On("GetLedger", mock.Anything, mock.Anything).
		Return(createLedgerCloseMeta(1), nil)

	go func() {
		<-time.After(time.Second * 1)
		cancel()
	}()

	go func() {
		ch := exporter.GetMetaArchiveChannel()
		for i := 0; i < 127; i++ {
			<-ch
		}
	}()

	err := exporter.Run(ctx, 0, 255)
	assert.EqualError(s.T(), err, "failed to add ledger 128: context canceled")

}

func (s *ExportManagerSuite) TestRunWithCanceledContext() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter := NewExportManager(config, &s.mockBackend)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := exporter.Run(ctx, 1, 10)
	assert.EqualError(s.T(), err, "context canceled")
}

func (s *ExportManagerSuite) TestAddLedgerCloseMeta() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter := NewExportManager(config, &s.mockBackend)
	objectCh := exporter.GetMetaArchiveChannel()
	expectedkeys := set.NewSet[string](10)
	actualKeys := set.NewSet[string](10)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range objectCh {
			actualKeys.Add(v.objectKey)
		}
	}()

	start := uint32(0)
	end := uint32(255)
	for i := start; i <= end; i++ {
		assert.NoError(s.T(), exporter.AddLedgerCloseMeta(context.Background(), createLedgerCloseMeta(i)))

		key, err := GetObjectKeyFromSequenceNumber(config, i)
		assert.NoError(s.T(), err)
		expectedkeys.Add(key)
	}

	close(objectCh)
	wg.Wait()
	assert.Equal(s.T(), expectedkeys, actualKeys)
}

func (s *ExportManagerSuite) TestAddLedgerCloseMetaContextCancel() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter := NewExportManager(config, &s.mockBackend)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Second * 1)
		cancel()
	}()

	assert.NoError(s.T(), exporter.AddLedgerCloseMeta(ctx, createLedgerCloseMeta(1)))
	err := exporter.AddLedgerCloseMeta(ctx, createLedgerCloseMeta(2))
	assert.EqualError(s.T(), err, "context canceled")
}
