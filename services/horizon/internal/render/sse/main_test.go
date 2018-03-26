package sse

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stellar/go/support/test"
	"github.com/stretchr/testify/assert"
)

func TestSsePackage(t *testing.T) {
	ctx, log := test.ContextWithLogBuffer()

	t.Run("sse.WriteEvent responses", func(t *testing.T) {
		expectations := []struct {
			Event     Event
			Substring string
		}{
			{Event{Data: "test"}, "data: \"test\"\n\n"},
			{Event{ID: "1", Data: "test"}, "id: 1\n"},
			{Event{Retry: 1000, Data: "test"}, "retry: 1000\n"},
			{Event{Error: errors.New("busted")}, "event: err\ndata: busted\n\n"},
			{Event{Event: "test", Data: "test"}, "event: test\ndata: \"test\"\n\n"},
			{Event{Event: "test", Data: "test"}, "event: test\ndata: \"test\"\n\n"},
			{Event{Comment: "I'm a comment"}, ": I'm a comment\n\n"},
		}

		for _, e := range expectations {
			w := httptest.NewRecorder()
			WriteEvent(ctx, w, e.Event)
			assert.Contains(t, w.Body.String(), e.Substring)
		}
	})

	t.Run("sse.WriteEvent logging", func(t *testing.T) {
		w := httptest.NewRecorder()
		WriteEvent(ctx, w, Event{Error: errors.New("busted")})
		assert.Contains(t, log.String(), "level=error")
		assert.Contains(t, log.String(), "busted")
	})
}
