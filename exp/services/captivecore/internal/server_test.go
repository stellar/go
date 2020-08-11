package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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
}

func (s *ServerTestSuite) SetupTest() {
	s.ledgerBackend = &ledgerbackend.MockDatabaseBackend{}
	s.api = NewCaptiveCoreAPI(s.ledgerBackend, log.New())
	s.handler = Handler(s.api)
}

func (s *ServerTestSuite) TearDownTest() {
	s.ledgerBackend.AssertExpectations(s.T())
}

func (s *ServerTestSuite) TestLatestSequence() {
	s.api.activeRequest.valid = true
	s.api.activeRequest.ready = true

	expectedSeq := uint32(100)
	s.ledgerBackend.On("GetLatestLedgerSequence").Return(expectedSeq, nil).Once()

	req := httptest.NewRequest("GET", "/latest-sequence", nil)
	w := httptest.NewRecorder()

	s.handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	s.Assert().NoError(err)

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var parsed LatestLedgerSequenceResponse
	s.Assert().NoError(json.Unmarshal(body, &parsed))
	s.Assert().Equal(LatestLedgerSequenceResponse{Sequence: expectedSeq}, parsed)
}

func (s *ServerTestSuite) TestLatestSequenceError() {
	s.api.activeRequest.valid = true
	s.api.activeRequest.ready = true

	s.ledgerBackend.On("GetLatestLedgerSequence").Return(uint32(100), fmt.Errorf("test error")).Once()

	req := httptest.NewRequest("GET", "/latest-sequence", nil)
	w := httptest.NewRecorder()

	s.handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	s.Assert().NoError(err)

	s.Assert().Equal(http.StatusInternalServerError, resp.StatusCode)
	s.Assert().Equal("test error", string(body))
}

func (s *ServerTestSuite) testPrepareRange(ledgerRange ledgerbackend.Range) {
	s.ledgerBackend.On("PrepareRange", ledgerRange).
		Return(nil).Once()

	body, err := json.Marshal(ledgerRange)
	s.Assert().NoError(err)

	req := httptest.NewRequest("POST", "/prepare-range", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	s.Assert().NoError(err)

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var parsed RangeResponse
	s.Assert().NoError(json.Unmarshal(body, &parsed))
	s.Assert().Equal(ledgerRange, parsed.LedgerRange)
	s.Assert().False(parsed.Ready)
	s.api.wg.Wait()
}

func (s *ServerTestSuite) TestPrepareBoundedRange() {
	s.testPrepareRange(ledgerbackend.BoundedRange(10, 30))
}

func (s *ServerTestSuite) TestPrepareUnboundedRange() {
	s.testPrepareRange(ledgerbackend.UnboundedRange(100))
}

func (s *ServerTestSuite) TestPrepareError() {
	s.ledgerBackend.On("Close").Return(nil).Once()
	s.api.Shutdown()

	body, err := json.Marshal(ledgerbackend.UnboundedRange(100))
	s.Assert().NoError(err)

	req := httptest.NewRequest("POST", "/prepare-range", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err = ioutil.ReadAll(resp.Body)
	s.Assert().NoError(err)

	s.Assert().Equal(http.StatusInternalServerError, resp.StatusCode)
	s.Assert().Equal("Cannot prepare range when shut down", string(body))
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

	req := httptest.NewRequest("GET", "/ledger/64", nil)
	w := httptest.NewRecorder()

	s.handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	s.Assert().NoError(err)

	s.Assert().Equal(http.StatusInternalServerError, resp.StatusCode)
	s.Assert().Equal("test error", string(body))
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

	req := httptest.NewRequest("GET", "/ledger/64", nil)
	w := httptest.NewRecorder()

	s.handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	s.Assert().NoError(err)

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var parsed LedgerResponse
	s.Assert().NoError(json.Unmarshal(body, &parsed))
	s.Assert().Equal(LedgerResponse{
		Present: true,
		Ledger:  Base64Ledger(expectedLedger),
	}, parsed)
}
