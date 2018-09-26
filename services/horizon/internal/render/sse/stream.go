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
	Init()
	Send(Event)
	SentCount() int
	Done()
	SetLimit(limit int)
	SetHeartbeatInterval(interval time.Duration)
	IsDone() bool
	Err(error)
}

// NewStream creates a new stream against the provided response writer.
func NewStream(ctx context.Context, w http.ResponseWriter, r *http.Request) Stream {
	result := &stream{
		ctx:      ctx,
		r:        r,
		interval: heartbeatInterval,
		w:        w,
	}

	return result
}

const heartbeatInterval = 10 * time.Second

type stream struct {
	ctx      context.Context
	r        *http.Request
	interval time.Duration // How often to send a heartbeat

	initSync sync.Once  // Variable to ensure that Init only writes the preamble once.
	mu       sync.Mutex // Mutex protects the following fields
	w        http.ResponseWriter
	done     bool
	sent     int
	limit    int
}

// Go routine that periodically sends a comment message to keep the connection
// alive.
func (s *stream) sendHeartbeats() {
	for {
		time.Sleep(s.interval)
		s.mu.Lock()
		if s.IsDone() {
			s.mu.Unlock()
			return
		}
		WriteHeartbeat(s.w)
		s.mu.Unlock()
	}
}

// Init function is only executed once. It writes the preamble event which includes the HTTP response code and a
// hello message. This should be called before any method that writes to the client to ensure that the preamble
// has been sent first.
func (s *stream) Init() {
	s.initSync.Do(func() {
		ok := WritePreamble(s.ctx, s.w)
		if !ok {
			s.done = true
		}
		// Start the go routine that sends heartbeats at regular intervals
		go s.sendHeartbeats()
	})
}

func (s *stream) Send(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Init()
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

func (s *stream) SetHeartbeatInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interval
}

func (s *stream) Done() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Init()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Init()
	WriteEvent(s.ctx, s.w, Event{Error: err})
	s.done = true
}
