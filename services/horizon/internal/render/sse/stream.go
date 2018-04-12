package sse

import (
	"context"
	"net/http"
	"time"
)

// Stream represents an output stream that data can be written to
type Stream interface {
	TrySendHeartbeat()
	Send(Event)
	SentCount() int
	Done()
	SetLimit(limit int)
	IsDone() bool
	Err(error)
}

// NewStream creates a new stream against the provided response writer
func NewStream(ctx context.Context, w http.ResponseWriter, r *http.Request) Stream {
	result := &stream{ctx, w, r, false, 0, 0, time.Now()}
	result.init()
	return result
}

type stream struct {
	ctx         context.Context
	w           http.ResponseWriter
	r           *http.Request
	done        bool
	sent        int
	limit       int
	lastWriteAt time.Time
}

func (s *stream) Send(e Event) {
	WriteEvent(s.ctx, s.w, e)
	s.lastWriteAt = time.Now()
	s.sent++
}

// TrySendHeartbeat will send
func (s *stream) TrySendHeartbeat() {

	if time.Since(s.lastWriteAt) < HeartbeatDelay {
		return
	}

	WriteHeartbeat(s.ctx, s.w)
	s.lastWriteAt = time.Now()
}

func (s *stream) SentCount() int {
	return s.sent
}

func (s *stream) SetLimit(limit int) {
	s.limit = limit
}

func (s *stream) Done() {
	WriteEvent(s.ctx, s.w, goodbyeEvent)
	s.done = true
}

func (s *stream) IsDone() bool {
	if s.limit == 0 {
		return s.done
	}

	return s.done || s.sent >= s.limit
}

func (s *stream) Err(err error) {
	WriteEvent(s.ctx, s.w, Event{Error: err})
	s.done = true
}

func (s *stream) init() {
	ok := WritePreamble(s.ctx, s.w)
	if !ok {
		s.done = true
	}

	return
}
