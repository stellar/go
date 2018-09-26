package sse

import (
	"context"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
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

func (suite *StreamTestSuite) SetupTest() {
	suite.ctx, _ = test.ContextWithLogBuffer()
	suite.w = httptest.NewRecorder()
	suite.stream = NewStream(suite.ctx, suite.w, nil)
}

func TestStream_Send(t *testing.T) {
	// Before sending,
	suite.S
}

// Tests that heartbeat events are sent by Stream.
func TestStream_SendHeartbeats(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()
	w := httptest.NewRecorder()
	stream := NewStream(ctx, w).(*stream)
	// Set heartbeat interval to a low value for testing.
	stream.interval = 500 * time.Millisecond
	stream.Init()
	// Wait long enough for heartbeat to send
	time.Sleep(time.Second)
	suite.stream.Done()
	assert.Contains(suite.T(), suite.w.Body.String(), ":heartbeat\n")
}

// Runs the test suite.
func TestStreamTestSuite(t *testing.T) {
	suite.Run(t, new(StreamTestSuite))
}
