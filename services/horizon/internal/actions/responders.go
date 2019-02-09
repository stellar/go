package actions

import "github.com/stellar/go/services/horizon/internal/render/sse"

// JSONer implementors can respond to a request whose response type was negotiated
// to be MimeHal or MimeJSON.
type JSONer interface {
	JSON() error
}

// Rawer implementors can respond to a request whose response type was negotiated
// to be MimeRaw.
type Rawer interface {
	Raw() error
}

// SSE implementors can respond to a request whose response type was negotiated
// to be MimeEventStream.
type SSE interface {
	SSE(*sse.Stream) error
}

// SingleObjectStreamer implementors can respond to a request whose response
// type was negotiated to be MimeEventStream. A SingleObjectStreamer loads an
// object whenever a ledger is closed.
type SingleObjectStreamer interface {
	LoadEvent() (sse.Event, error)
}
