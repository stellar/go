package sse

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStream_TrySendHeartBeat(t *testing.T) {
	hbCount := func(w *httptest.ResponseRecorder) int {
		return strings.Count(w.Body.String(), "bu-bump")
	}

	t.Run("delay between heartbeats", func(t *testing.T) {
		w := httptest.NewRecorder()
		s := NewStream(context.Background(), w, nil)

		s.TrySendHeartbeat()
		assert.Equal(t, 0, hbCount(w), "unexpected heartbeat: #1")

		time.Sleep(HeartbeatDelay)
		s.TrySendHeartbeat()
		assert.Equal(t, 1, hbCount(w), "initial heartbeat failed")

		time.Sleep(HeartbeatDelay / 2)
		s.TrySendHeartbeat()
		assert.Equal(t, 1, hbCount(w), "unexpected heartbeat: #2")

		time.Sleep(HeartbeatDelay / 2)
		s.TrySendHeartbeat()
		assert.Equal(t, 2, hbCount(w), "2nd expected heartbeat not seen")
	})

	t.Run("delay after message", func(t *testing.T) {
		w := httptest.NewRecorder()
		s := NewStream(context.Background(), w, nil)

		s.Send(Event{Data: "I'm real!"})
		assert.Equal(t, 0, hbCount(w))
		s.TrySendHeartbeat()
		assert.Equal(t, 0, hbCount(w))

		time.Sleep(HeartbeatDelay)
		s.TrySendHeartbeat()
		assert.Equal(t, 1, hbCount(w), "expected heartbeat not seen")
	})
}
