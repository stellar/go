package sse

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stellar/go/support/test"
	"github.com/stretchr/testify/assert"
)

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
	stream.Done()
	assert.Contains(t, w.Body.String(), "\n:heartbeat")
}
