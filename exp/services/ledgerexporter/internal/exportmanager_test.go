package ledgerexporter

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/collections/set"
	"github.com/stretchr/testify/require"
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

func (s *ExportManagerSuite) TestInvalidExportConfig() {
	config := ExporterConfig{LedgersPerFile: 0, FilesPerPartition: 10}
	_, err := NewExportManager(config, &s.mockBackend)
	require.Error(s.T(), err)
}

func (s *ExportManagerSuite) TestRun() {
	config := ExporterConfig{LedgersPerFile: 64, FilesPerPartition: 10}
	exporter, err := NewExportManager(config, &s.mockBackend)
	require.NoError(s.T(), err)

	start := uint32(0)
	end := uint32(255)
	expectedKeys := set.NewSet[string](10)
	for i := start; i <= end; i++ {
		s.mockBackend.On("GetLedger", s.ctx, i).
			Return(createLedgerCloseMeta(i), nil)
		key := config.GetObjectKeyFromSequenceNumber(i)
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

	err = exporter.Run(s.ctx, start, end)
	require.NoError(s.T(), err)

	wg.Wait()

	require.Equal(s.T(), expectedKeys, actualKeys)
}

func (s *ExportManagerSuite) TestRunContextCancel() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 1}
	exporter, err := NewExportManager(config, &s.mockBackend)
	require.NoError(s.T(), err)
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

	err = exporter.Run(ctx, 0, 255)
	require.EqualError(s.T(), err, "failed to add ledgerCloseMeta for ledger 128: context canceled")

}

func (s *ExportManagerSuite) TestRunWithCanceledContext() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter, err := NewExportManager(config, &s.mockBackend)
	require.NoError(s.T(), err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = exporter.Run(ctx, 1, 10)
	require.EqualError(s.T(), err, "context canceled")
}

func (s *ExportManagerSuite) TestGetObjectKeyFromSequenceNumber() {
	testCases := []struct {
		filesPerPartition uint32
		ledgerSeq         uint32
		ledgersPerFile    uint32
		expectedKey       string
	}{
		{0, 5, 1, "5.xdr.gz"},
		{0, 5, 10, "0-9.xdr.gz"},
		{2, 10, 100, "0-199/0-99.xdr.gz"},
		{2, 150, 50, "100-199/150-199.xdr.gz"},
		{2, 300, 200, "0-399/200-399.xdr.gz"},
		{2, 1, 1, "0-1/1.xdr.gz"},
		{4, 10, 100, "0-399/0-99.xdr.gz"},
		{4, 250, 50, "200-399/250-299.xdr.gz"},
		{1, 300, 200, "200-399.xdr.gz"},
		{1, 1, 1, "1.xdr.gz"},
	}

	for _, tc := range testCases {
		s.T().Run(fmt.Sprintf("LedgerSeq-%d-LedgersPerFile-%d", tc.ledgerSeq, tc.ledgersPerFile), func(t *testing.T) {
			config := ExporterConfig{FilesPerPartition: tc.filesPerPartition, LedgersPerFile: tc.ledgersPerFile}
			key := config.GetObjectKeyFromSequenceNumber(tc.ledgerSeq)
			require.Equal(t, tc.expectedKey, key)
		})
	}
}

func (s *ExportManagerSuite) TestAddLedgerCloseMeta() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter, err := NewExportManager(config, &s.mockBackend)
	require.NoError(s.T(), err)
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
		require.NoError(s.T(), exporter.AddLedgerCloseMeta(context.Background(), createLedgerCloseMeta(i)))
		key := config.GetObjectKeyFromSequenceNumber(i)
		expectedkeys.Add(key)
	}

	close(objectCh)
	wg.Wait()
	require.Equal(s.T(), expectedkeys, actualKeys)
}

func (s *ExportManagerSuite) TestAddLedgerCloseMetaContextCancel() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter, err := NewExportManager(config, &s.mockBackend)
	require.NoError(s.T(), err)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Second * 1)
		cancel()
	}()

	require.NoError(s.T(), exporter.AddLedgerCloseMeta(ctx, createLedgerCloseMeta(1)))
	err = exporter.AddLedgerCloseMeta(ctx, createLedgerCloseMeta(2))
	require.EqualError(s.T(), err, "context canceled")
}

func (s *ExportManagerSuite) TestAddLedgerCloseMetaKeyMismatch() {
	config := ExporterConfig{LedgersPerFile: 10, FilesPerPartition: 1}
	exporter, err := NewExportManager(config, &s.mockBackend)
	require.NoError(s.T(), err)

	require.NoError(s.T(), exporter.AddLedgerCloseMeta(context.Background(), createLedgerCloseMeta(16)))
	require.EqualError(s.T(), exporter.AddLedgerCloseMeta(context.Background(), createLedgerCloseMeta(21)),
		"Current meta archive object key mismatch")
}
