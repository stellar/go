package sse

import (
	"context"
	"net/http"
	"sync"
	"time"
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

// NewStream creates a new stream against the provided response writer. It immediately sends a
// preamble response with the appropriate HTTP headers for SSE. It then starts a goroutine
// for sending periodic heartbeats to keep the connection alive even in the absence of events.
func NewStream(ctx context.Context, w http.ResponseWriter, r *http.Request) Stream {
	result := &stream{
		ctx:   ctx,
		r:     r,
		interval: heartbeatInterval,
		w:     w,
	}
	result.mu.Lock()
	defer result.mu.Unlock()
	ok := WritePreamble(result.ctx, result.w)
	if !ok {
		result.done = true
		return result
	}
	// Start the go routine that sends heartbeats at regular intervals
	go result.sendHeartbeats()
	return result
}

const heartbeatInterval = 10*time.Second

type stream struct {
	ctx context.Context
	r   *http.Request
	interval time.Duration	// How often to send a heartbeat

	mu    sync.Mutex // Mutex protects the following fields.
	w     http.ResponseWriter
	done  bool
	sent  int
	limit int
}

// Go routine that periodically sends a comment message to keep the connection
// alive.
func (s *stream) sendHeartbeats() {
	for {
		time.Sleep(heartbeatInterval)
		if s.IsDone() {
			return
		}
		s.mu.Lock()
		WriteHeartbeat(s.w)
		s.mu.Unlock()
	}
}

func (s *stream) Send(e Event) {
	s.mu.Lock()
	WriteEvent(s.ctx, s.w, e)
	s.sent++
	s.mu.Unlock()
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
