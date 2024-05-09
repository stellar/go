package ledgerexporter

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/support/datastore"
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

func (s *ExportManagerSuite) TestInvalidExportConfig() {
	config := datastore.LedgerBatchConfig{LedgersPerFile: 0, FilesPerPartition: 10, FileSuffix: ".xdr.gz"}
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	_, err := NewExportManager(config, &s.mockBackend, queue, registry)
	require.Error(s.T(), err)
}

func (s *ExportManagerSuite) TestRun() {
	config := datastore.LedgerBatchConfig{LedgersPerFile: 64, FilesPerPartition: 10, FileSuffix: ".xdr.gz"}
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	exporter, err := NewExportManager(config, &s.mockBackend, queue, registry)
	require.NoError(s.T(), err)

	start := uint32(0)
	end := uint32(255)
	expectedKeys := set.NewSet[string](10)
	for i := start; i <= end; i++ {
		s.mockBackend.On("GetLedger", s.ctx, i).
			Return(datastore.CreateLedgerCloseMeta(i), nil)
		key := config.GetObjectKeyFromSequenceNumber(i)
		expectedKeys.Add(key)
	}

	actualKeys := set.NewSet[string](10)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			v, ok, dqErr := queue.Dequeue(s.ctx)
			s.Assert().NoError(dqErr)
			if !ok {
				break
			}
			actualKeys.Add(v.ObjectKey)
		}
	}()

	err = exporter.Run(s.ctx, start, end)
	require.NoError(s.T(), err)

	wg.Wait()

	require.Equal(s.T(), expectedKeys, actualKeys)
	require.Equal(
		s.T(),
		float64(255),
		getMetricValue(exporter.latestLedgerMetric.With(
			prometheus.Labels{
				"start_ledger": "0",
				"end_ledger":   "255",
			}),
		).GetGauge().GetValue(),
	)
}

func (s *ExportManagerSuite) TestRunContextCancel() {
	config := datastore.LedgerBatchConfig{LedgersPerFile: 1, FilesPerPartition: 1, FileSuffix: ".xdr.gz"}
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	exporter, err := NewExportManager(config, &s.mockBackend, queue, registry)
	require.NoError(s.T(), err)
	ctx, cancel := context.WithCancel(context.Background())

	s.mockBackend.On("GetLedger", mock.Anything, mock.Anything).
		Return(datastore.CreateLedgerCloseMeta(1), nil)

	go func() {
		<-time.After(time.Second * 1)
		cancel()
	}()

	go func() {
		for i := 0; i < 127; i++ {
			_, ok, dqErr := queue.Dequeue(s.ctx)
			s.Assert().NoError(dqErr)
			s.Assert().True(ok)
		}
	}()

	err = exporter.Run(ctx, 0, 255)
	require.EqualError(s.T(), err, "failed to add ledgerCloseMeta for ledger 128: context canceled")

}

func (s *ExportManagerSuite) TestRunWithCanceledContext() {
	config := datastore.LedgerBatchConfig{LedgersPerFile: 1, FilesPerPartition: 10, FileSuffix: ".xdr.gz"}
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	exporter, err := NewExportManager(config, &s.mockBackend, queue, registry)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = exporter.Run(ctx, 1, 10)
	require.EqualError(s.T(), err, "context canceled")
}

func (s *ExportManagerSuite) TestAddLedgerCloseMeta() {
	config := datastore.LedgerBatchConfig{LedgersPerFile: 1, FilesPerPartition: 10, FileSuffix: ".xdr.gz"}
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	exporter, err := NewExportManager(config, &s.mockBackend, queue, registry)
	require.NoError(s.T(), err)

	expectedKeys := set.NewSet[string](10)
	actualKeys := set.NewSet[string](10)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			v, ok, err := queue.Dequeue(s.ctx)
			s.Assert().NoError(err)
			if !ok {
				break
			}
			actualKeys.Add(v.ObjectKey)
		}
	}()

	start := uint32(0)
	end := uint32(255)
	for i := start; i <= end; i++ {
		require.NoError(s.T(), exporter.AddLedgerCloseMeta(context.Background(), datastore.CreateLedgerCloseMeta(i)))

		key := config.GetObjectKeyFromSequenceNumber(i)
		expectedKeys.Add(key)
	}

	queue.Close()
	wg.Wait()
	require.Equal(s.T(), expectedKeys, actualKeys)
}

func (s *ExportManagerSuite) TestAddLedgerCloseMetaContextCancel() {
	config := datastore.LedgerBatchConfig{LedgersPerFile: 1, FilesPerPartition: 10, FileSuffix: ".xdr.gz"}
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	exporter, err := NewExportManager(config, &s.mockBackend, queue, registry)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Second * 1)
		cancel()
	}()

	require.NoError(s.T(), exporter.AddLedgerCloseMeta(ctx, datastore.CreateLedgerCloseMeta(1)))
	err = exporter.AddLedgerCloseMeta(ctx, datastore.CreateLedgerCloseMeta(2))
	require.EqualError(s.T(), err, "context canceled")
}

func (s *ExportManagerSuite) TestAddLedgerCloseMetaKeyMismatch() {
	config := datastore.LedgerBatchConfig{LedgersPerFile: 10, FilesPerPartition: 1, FileSuffix: ".xdr.gz"}
	registry := prometheus.NewRegistry()
	queue := NewUploadQueue(1, registry)
	exporter, err := NewExportManager(config, &s.mockBackend, queue, registry)
	require.NoError(s.T(), err)

	require.NoError(s.T(), exporter.AddLedgerCloseMeta(context.Background(), datastore.CreateLedgerCloseMeta(16)))
	require.EqualError(s.T(), exporter.AddLedgerCloseMeta(context.Background(), datastore.CreateLedgerCloseMeta(21)),
		"Current meta archive object key mismatch")
}
