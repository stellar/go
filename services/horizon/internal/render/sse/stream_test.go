package sse

import (
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stellar/go/support/test"
	"github.com/stretchr/testify/assert"
)

type StreamTestSuite struct {
	suite.Suite
	ctx context.Context
	w *httptest.ResponseRecorder
	stream Stream
}

func (suite *StreamTestSuite) checkHeadersAndPreamble() {
	assert.Equal(suite.T(), "text/event-stream; charset=utf-8", suite.w.Header().Get("Content-Type"))
	assert.Equal(suite.T(), "no-cache", suite.w.Header().Get("Cache-Control"))
	assert.Equal(suite.T(), "keep-alive", suite.w.Header().Get("Connection"))
	assert.Equal(suite.T(), "*", suite.w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(suite.T(), 200, suite.w.Code)
	assert.Contains(suite.T(), suite.w.Body.String(), "retry: 1000\nevent: open\ndata: \"hello\"\n\n")
}

func (suite *StreamTestSuite) SetupTest() {
	suite.ctx, _ = test.ContextWithLogBuffer()
	suite.w = httptest.NewRecorder()
	suite.stream = NewStream(suite.ctx, suite.w, nil)
}

// Tests that the stream sends the preamble before any events and that events are correctly sent.
func (suite *StreamTestSuite) TestStream_Send() {
	e := Event{Data:"test message"}
	suite.stream.Send(e)
	suite.stream.Done()
	// Before sending, it should have sent the preamble first and set the headers.
	suite.checkHeadersAndPreamble()
	// Now check that the data got written
	assert.Contains(suite.T(), suite.w.Body.String(), "test message")
	assert.Equal(suite.T(), 1, suite.stream.SentCount())
}

// Tests that heartbeat events are sent by Stream.
func (suite *StreamTestSuite) TestStream_SendHeartbeats() {
	// Set heartbeat interval to a low value for testing.
	suite.stream.(*stream).interval = 500*time.Millisecond
	suite.stream.Init()
	// Wait long enough for heartbeat to send
	time.Sleep(time.Second)
	suite.stream.Done()
	assert.Contains(suite.T(), suite.w.Body.String(), ":heartbeat\n")
}

// Tests that Stream can send error events.
func (suite *StreamTestSuite) TestStream_Err() {
	err := errors.New("example error")
	suite.stream.Err(err)
	// Even if no events have been sent, Err should still send the preamble before the error event.
	suite.checkHeadersAndPreamble()
	assert.Contains(suite.T(), suite.w.Body.String(), "event: err\ndata: example error\n\n")
	assert.True(suite.T(), suite.stream.IsDone())
}

// Tests that SetLimit stops the stream after the limit has been reached.
func (suite *StreamTestSuite) TestStream_SetLimit() {
	suite.stream.SetLimit(3)
	// Send more than the limit
	for i := 0; i < 5; i++ {
		message := "test message " + strconv.Itoa(i)
		suite.stream.Send(Event{Data:message})
	}
	assert.True(suite.T(), suite.stream.IsDone())
	println(suite.w.Body.String())
}
// Runs the test suite.
func TestStreamTestSuite(t *testing.T) {
	suite.Run(t, new(StreamTestSuite))
}
