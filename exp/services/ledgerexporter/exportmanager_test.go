package main

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
	_ "github.com/stellar/go/xdr"
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
	testChan    chan *LedgerCloseMetaObject
}

func (s *ExportManagerSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockBackend = ledgerbackend.MockDatabaseBackend{}
	s.testChan = make(chan *LedgerCloseMetaObject)
}

func (s *ExportManagerSuite) TearDownTest() {
}

func (s *ExportManagerSuite) TestRun() {
	config := ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	exporter := NewExportManager(config, &s.mockBackend, s.testChan)

	start := 0
	end := 20
	for i := start; i <= end; i++ {
		s.mockBackend.On("GetLedger", s.ctx, uint32(i)).Return(
			xdr.LedgerCloseMeta{
				V0: &xdr.LedgerCloseMetaV0{
					LedgerHeader: xdr.LedgerHeaderHistoryEntry{
						Header: xdr.LedgerHeader{
							LedgerSeq:      xdr.Uint32(i),
							LedgerVersion:  xdr.Uint32(20),
							BucketListHash: xdr.Hash{1, 2, 3},
						},
					},
				},
			}, nil)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range s.testChan {
			//TODO: Add asserts for objectkeys
			fmt.Println(v.objectKey)
		}
	}()

	exporter.Run(s.ctx, uint32(start), uint32(end))
	wg.Wait()
}
