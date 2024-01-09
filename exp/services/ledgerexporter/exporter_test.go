package main

import (
	"context"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/storage"
	"github.com/stellar/go/xdr"
	_ "github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestExporterSuite(t *testing.T) {
	suite.Run(t, new(ExporterSuite))
}

// ExporterSuite is a test suite for the Exporter.
type ExporterSuite struct {
	suite.Suite
	ctx         context.Context
	exporter    *Exporter
	backend     ledgerbackend.MockDatabaseBackend
	destination storage.MockStorage
	config      ExporterConfig
}

func (s *ExporterSuite) SetupTest() {
	s.ctx = context.Background()
	s.config = ExporterConfig{LedgersPerFile: 1, FilesPerPartition: 10}
	s.backend = ledgerbackend.MockDatabaseBackend{}
	s.destination = storage.MockStorage{}
	s.exporter = NewExporter(s.config, &s.destination, &s.backend)
}

func (s *ExporterSuite) TearDownTest() {
}

func (s *ExporterSuite) TestRun() {
	start := 1
	end := 20
	for i := start; i <= end; i++ {
		s.backend.On("GetLedger", s.ctx, uint32(i)).Return(
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
		s.destination.On("PutFile", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	}
	s.exporter.Run(s.ctx, uint32(start), uint32(end))
	s.backend.AssertExpectations(s.T())
}
