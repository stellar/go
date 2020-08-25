package internal

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

type ServerTestSuite struct {
	suite.Suite
	ledgerBackend *ledgerbackend.MockDatabaseBackend
	api           CaptiveCoreAPI
	handler       http.Handler
	server        *httptest.Server
	client        ledgerbackend.RemoteCaptiveStellarCore
}

func (s *ServerTestSuite) SetupTest() {
	s.ledgerBackend = &ledgerbackend.MockDatabaseBackend{}
	s.api = NewCaptiveCoreAPI(s.ledgerBackend, log.New())
	s.handler = Handler(s.api)
	s.server = httptest.NewServer(s.handler)
	var err error
	s.client, err = ledgerbackend.NewRemoteCaptive(
		s.server.URL,
		ledgerbackend.PrepareRangePollInterval(time.Millisecond),
	)
	s.Assert().NoError(err)
}

func (s *ServerTestSuite) TearDownTest() {
	s.ledgerBackend.AssertExpectations(s.T())
	s.server.Close()
	s.client.Close()
}

func (s *ServerTestSuite) TestLatestSequence() {
	s.api.activeRequest.valid = true
	s.api.activeRequest.ready = true

	expectedSeq := uint32(100)
	s.ledgerBackend.On("GetLatestLedgerSequence").Return(expectedSeq, nil).Once()

	seq, err := s.client.GetLatestLedgerSequence()
	s.Assert().NoError(err)
	s.Assert().Equal(expectedSeq, seq)
}

func (s *ServerTestSuite) TestLatestSequenceError() {
	s.api.activeRequest.valid = true
	s.api.activeRequest.ready = true

	s.ledgerBackend.On("GetLatestLedgerSequence").Return(uint32(100), fmt.Errorf("test error")).Once()

	_, err := s.client.GetLatestLedgerSequence()
	s.Assert().EqualError(err, "test error")
}

func (s *ServerTestSuite) TestPrepareBoundedRange() {
	ledgerRange := ledgerbackend.BoundedRange(10, 30)
	s.ledgerBackend.On("PrepareRange", ledgerRange).
		Return(nil).Once()

	s.Assert().NoError(s.client.PrepareRange(ledgerRange))
	s.Assert().True(s.api.activeRequest.ready)

	prepared, err := s.client.IsPrepared(ledgerRange)
	s.Assert().NoError(err)
	s.Assert().True(prepared)
}

func (s *ServerTestSuite) TestPrepareUnboundedRange() {
	ledgerRange := ledgerbackend.UnboundedRange(100)
	s.ledgerBackend.On("PrepareRange", ledgerRange).
		Return(nil).Once()

	s.Assert().NoError(s.client.PrepareRange(ledgerRange))
	s.Assert().True(s.api.activeRequest.ready)

	prepared, err := s.client.IsPrepared(ledgerRange)
	s.Assert().NoError(err)
	s.Assert().True(prepared)
}

func (s *ServerTestSuite) TestPrepareError() {
	s.ledgerBackend.On("Close").Return(nil).Once()
	s.api.Shutdown()

	s.Assert().EqualError(
		s.client.PrepareRange(ledgerbackend.UnboundedRange(100)),
		"Cannot prepare range when shut down",
	)
}

func (s *ServerTestSuite) TestGetLedgerInvalidSequence() {
	req := httptest.NewRequest("GET", "/ledger/abcdef", nil)
	w := httptest.NewRecorder()

	s.handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	s.Assert().NoError(err)

	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
	s.Assert().Equal("path params could not be parsed: schema: error converting value for \"sequence\"", string(body))
}

func (s *ServerTestSuite) TestGetLedgerError() {
	s.api.activeRequest.valid = true
	s.api.activeRequest.ready = true

	expectedErr := fmt.Errorf("test error")
	s.ledgerBackend.On("GetLedger", uint32(64)).
		Return(false, xdr.LedgerCloseMeta{}, expectedErr).Once()

	_, _, err := s.client.GetLedger(64)
	s.Assert().EqualError(err, "test error")
}

func (s *ServerTestSuite) TestGetLedgerSucceeds() {
	s.api.activeRequest.valid = true
	s.api.activeRequest.ready = true

	expectedLedger := xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: 64,
				},
			},
		},
	}
	s.ledgerBackend.On("GetLedger", uint32(64)).
		Return(true, expectedLedger, nil).Once()

	present, ledger, err := s.client.GetLedger(64)
	s.Assert().NoError(err)
	s.Assert().True(present)
	s.Assert().Equal(expectedLedger, ledger)
}
