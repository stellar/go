package exporter

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/collections/set"
	"github.com/stellar/go/xdr"
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
}

func (s *ExportManagerSuite) TestRun() {
	config := ExporterConfig{LedgersPerFile: 64, FilesPerPartition: 10}
	exporter := NewExportManager(config, &s.mockBackend)
	expectedObjectkeys := set.NewSet[string](10)
	start := 2
	end := 255
	for i := start; i <= end; i++ {
		s.mockBackend.On("GetLedger", s.ctx, uint32(i)).Return(
			xdr.LedgerCloseMeta{
				V0: &xdr.LedgerCloseMetaV0{
					LedgerHeader: xdr.LedgerHeaderHistoryEntry{
						Header: xdr.LedgerHeader{
							LedgerSeq: xdr.Uint32(i),
						},
					},
				},
			}, nil)

		key, _ := config.getObjectKey(uint32(i))
		expectedObjectkeys.Add(key)
	}

	actualObjectKeys := set.NewSet[string](10)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range exporter.GetExportObjectsChannel() {
			actualObjectKeys.Add(v.objectKey)
		}
	}()

	err := exporter.Run(s.ctx, uint32(start), uint32(end))
	assert.NoError(s.T(), err)

	wg.Wait()

	assert.Equal(s.T(), expectedObjectkeys, actualObjectKeys)
	s.mockBackend.AssertExpectations(s.T())
}

func (s *ExportManagerSuite) TestAddLedgerCloseMeta() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter := NewExportManager(config, &s.mockBackend)
	objectCh := exporter.GetExportObjectsChannel()
	expectedObjectkeys := set.NewSet[string](10)
	actualObjectKeys := set.NewSet[string](10)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range objectCh {
			actualObjectKeys.Add(v.objectKey)
		}
	}()

	start := 0
	end := 255
	for i := start; i <= end; i++ {
		err := exporter.AddLedgerCloseMeta(xdr.LedgerCloseMeta{
			V0: &xdr.LedgerCloseMetaV0{
				LedgerHeader: xdr.LedgerHeaderHistoryEntry{
					Header: xdr.LedgerHeader{
						LedgerSeq: xdr.Uint32(i),
					},
				},
			},
		})
		assert.NoError(s.T(), err)
		key, _ := config.getObjectKey(uint32(i))
		expectedObjectkeys.Add(key)
	}

	close(objectCh)
	wg.Wait()
	assert.Equal(s.T(), expectedObjectkeys, actualObjectKeys)
}

func (s *ExportManagerSuite) TestAddLedgerCloseMetaSequential() {
	config := ExporterConfig{LedgersPerFile: 10, FilesPerPartition: 1}
	exporter := NewExportManager(config, &s.mockBackend)

	// Add ledgers sequentially
	for i := 1; i <= 5; i++ {
		assert.NoError(s.T(),
			exporter.AddLedgerCloseMeta(xdr.LedgerCloseMeta{
				V0: &xdr.LedgerCloseMetaV0{
					LedgerHeader: xdr.LedgerHeaderHistoryEntry{
						Header: xdr.LedgerHeader{
							LedgerSeq: xdr.Uint32(i),
						},
					},
				},
			}))
	}

	// Add a ledger out of sequence
	err := exporter.AddLedgerCloseMeta(xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(7),
				},
			},
		},
	})

	expectedErrMsg := fmt.Sprintf("failed to add ledger to LedgerCloseMetaObject: ledgers must be added sequentially."+
		" Sequence number: %d, expected sequence number: %d", 7, 6)
	assert.EqualError(s.T(), err, expectedErrMsg)

}
