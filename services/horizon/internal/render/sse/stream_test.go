package sse

import (
	"context"
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stellar/go/support/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StreamTestSuite struct {
	suite.Suite
	ctx    context.Context
	w      *httptest.ResponseRecorder
	stream Stream
}

// Helper method to check that the preamble has been sent and all HTTP response headers are correctly set.
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
	suite.stream = NewStream(suite.ctx, suite.w)
}

// Tests that the stream sends the preamble before any events and that events are correctly sent.
func (suite *StreamTestSuite) TestStream_Send() {
	e := Event{Data: "test message"}
	suite.stream.Send(e)
	// Before sending, it should have sent the preamble first and set the headers.
	suite.checkHeadersAndPreamble()
	// Now check that the data got written
	assert.Contains(suite.T(), suite.w.Body.String(), "data: \"test message\"\n\n")
	suite.stream.Done()
	assert.Equal(suite.T(), 1, suite.stream.SentCount())
}

// Tests that heartbeat events are sent by Stream.
func (suite *StreamTestSuite) TestStream_SendHeartbeats() {
	// Set heartbeat interval to a low value for testing.
	suite.stream.(*stream).interval = 100 * time.Millisecond
	suite.stream.Init()
	for i := 0; i < 3; i++ {
		// Wait long enough for a heartbeat to send
		time.Sleep(110 * time.Millisecond)
		assert.True(suite.T(), suite.w.Flushed)
		occurrences := strings.Count(suite.w.Body.String(), ":heartbeat")
		assert.Equal(suite.T(), i + 1, occurrences)
	}
	suite.stream.Done()
}

// Tests that exceeding the send limit stops the heartbeat routine.
func (suite *StreamTestSuite) TestStream_HeartbeatsLimitExceeded() {
	// Set heartbeat interval to a low value for testing.
	suite.stream.(*stream).interval = 500 * time.Millisecond
	suite.stream.SetLimit(1)
	suite.stream.Init()
	// Send more messages than the limit allows
	for i := 0; i < 2; i++ {
		message := "test message " + strconv.Itoa(i)
		suite.stream.Send(Event{Data: message})
	}
	// Wait long enough so that a heartbeat event would have fired if the stream wasn't done.
	time.Sleep(time.Second)
	assert.True(suite.T(), suite.stream.IsDone())
	assert.NotContains(suite.T(), suite.w.Body.String(), "\n:heartbeat")
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

// Tests that SetLimit sets stream.done to true after the limit has been reached.
func (suite *StreamTestSuite) TestStream_SetLimit() {
	suite.stream.SetLimit(3)
	// Send more than the limit
	for i := 0; i < 5; i++ {
		message := "test message " + strconv.Itoa(i)
		suite.stream.Send(Event{Data: message})
	}
	assert.True(suite.T(), suite.stream.IsDone())
}

// Tests that SentCount reports the correct number.
func (suite *StreamTestSuite) TestStream_SentCount() {
	for i := 0; i < 5; i++ {
		message := "test message " + strconv.Itoa(i)
		suite.stream.Send(Event{Data: message})
	}
	assert.Equal(suite.T(), 5, suite.stream.SentCount())
	suite.stream.Err(errors.New("example error"))
	// Make sure that errors don't contribute to the send count
	assert.Equal(suite.T(), 5, suite.stream.SentCount())
}

// Tests that calling Done stops the heartbeat goroutine.
func (suite *StreamTestSuite) TestStream_Done() {
	suite.stream.(*stream).interval = 500 * time.Millisecond
	suite.stream.Init()
	suite.stream.Done()
	// Wait long enough so that a heartbeat event would have fired.
	time.Sleep(time.Second)
	assert.True(suite.T(), suite.stream.IsDone())
	assert.NotContains(suite.T(), suite.w.Body.String(), ":heartbeat")
}

// Runs the test suite.
func TestStreamTestSuite(t *testing.T) {
	suite.Run(t, new(StreamTestSuite))
}
