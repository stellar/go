package sse

import (
	"net/http"
	"sync"

	"github.com/r3labs/sse"
)

// AddressEvent is an event sent to address SSE stream.
type AddressEvent string

const (
	TransactionReceivedAddressEvent AddressEvent = "transaction_received"
	AccountCreatedAddressEvent      AddressEvent = "account_created"
	AccountCreditedAddressEvent     AddressEvent = "account_credited"
)

type Server struct {
	eventsServer *sse.Server
	initOnce     sync.Once
}

type ServerInterface interface {
	PublishEvent(address string, event AddressEvent, data []byte)
	CreateStream(address string)
	StreamExists(address string) bool
	HTTPHandler(w http.ResponseWriter, r *http.Request)
}

func (s *Server) init() {
	s.eventsServer = sse.New()
}

func (s *Server) PublishEvent(address string, event AddressEvent, data []byte) {
	s.initOnce.Do(s.init)

	// Create SSE stream if not exists
	if !s.eventsServer.StreamExists(address) {
		s.eventsServer.CreateStream(address)
	}

	// github.com/r3labs/sse does not send new lines - TODO create PR
	if data == nil {
		data = []byte("{}\n")
	} else {
		data = append(data, byte('\n'))
	}

	s.eventsServer.Publish(address, &sse.Event{
		Event: []byte(event),
		Data:  data,
	})
}

func (s *Server) CreateStream(address string) {
	s.initOnce.Do(s.init)
	s.eventsServer.CreateStream(address)
}

func (s *Server) StreamExists(address string) bool {
	s.initOnce.Do(s.init)
	return s.eventsServer.StreamExists(address)
}

func (s *Server) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	s.initOnce.Do(s.init)
	s.eventsServer.HTTPHandler(w, r)
}
