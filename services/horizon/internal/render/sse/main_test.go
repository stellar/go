package sse

import (
	"bytes"
	"context"
	"errors"
	"github.com/stellar/go/support/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"testing"
)

type SsePackageTestSuite struct {
	suite.Suite
	ctx context.Context
	log *bytes.Buffer
}

func (suite *SsePackageTestSuite) SetupTest() {
	suite.ctx, suite.log = test.ContextWithLogBuffer()
}

func (suite *SsePackageTestSuite) TestWriteEventOutput() {
	expectations := []struct {
		Event     Event
		Substring string
	}{
		{Event{Data: "test"}, "data: \"test\"\n\n"},
		{Event{ID: "1", Data: "test"}, "id: 1\n"},
		{Event{Retry: 1000, Data: "test"}, "retry: 1000\n"},
		{Event{Error: errors.New("busted")}, "event: err\ndata: busted\n\n"},
		{Event{Event: "test", Data: "test"}, "event: test\ndata: \"test\"\n\n"},
	}

	for _, e := range expectations {
		w := httptest.NewRecorder()
		WriteEvent(suite.ctx, w, e.Event)
		bodyString := w.Body.String()
		assert.Equal(suite.T(), e.Substring, bodyString)
	}
}

func (suite *SsePackageTestSuite) TestWriteEventLogs() {
	w := httptest.NewRecorder()
	WriteEvent(suite.ctx, w, Event{Error: errors.New("busted")})
	assert.Contains(suite.T(), suite.log.String(), "level=error")
	assert.Contains(suite.T(), suite.log.String(), "busted")
}

func TestSsePackageTestSuite(t *testing.T) {
	suite.Run(t, new(SsePackageTestSuite))
}
