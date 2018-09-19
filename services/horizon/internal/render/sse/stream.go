package sse

import (
	"context"
	"net/http"
	"sync"
)

// Stream represents an output stream that data can be written to.
// Its methods must be safe to call concurrently.
type Stream interface {
	Send(Event)
	SentCount() int
	Done()
	SetLimit(limit int)
	IsDone() bool
	Err(error)
}

// NewStream creates a new stream against the provided response writer
func NewStream(ctx context.Context, w http.ResponseWriter, r *http.Request) Stream {
	result := &stream{
		ctx:   ctx,
		r:     r,
		w:     w,
		done:  false,
		sent:  0,
		limit: 0,
	}
	return result
}

type stream struct {
	ctx context.Context
	r   *http.Request

	mu    sync.Mutex // Mutex protects the following fields
	w     http.ResponseWriter
	done  bool
	sent  int
	limit int
}

func (s *stream) Send(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sent == 0 {
		ok := WritePreamble(s.ctx, s.w)
		if !ok {
			s.done = true
			return
		}
	}

	WriteEvent(s.ctx, s.w, e)
	s.sent++
}

func (s *stream) SentCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sent
}

func (s *stream) SetLimit(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limit = limit
}

func (s *stream) Done() {
	s.mu.Lock()
	defer s.mu.Unlock()
	WriteEvent(s.ctx, s.w, goodbyeEvent)
	s.done = true
}

func (s *stream) IsDone() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.limit == 0 {
		return s.done
	}

	return s.done || s.sent >= s.limit
}

func (s *stream) Err(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	WriteEvent(s.ctx, s.w, Event{Error: err})
	s.done = true
}
