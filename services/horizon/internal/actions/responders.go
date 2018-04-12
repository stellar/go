package actions

import "github.com/stellar/go/services/horizon/internal/render/sse"

// JSON implementors can respond to a request whose response type was negotiated
// to be MimeHal or MimeJSON.
type JSON interface {
	JSON()
}

// Raw implementors can respond to a request whose response type was negotiated
// to be MimeRaw.
type Raw interface {
	Raw()
}

// SSE implementors can respond to a request whose response type was negotiated
// to be MimeEventStream.
type SSE interface {
	SSE(sse.Stream)
}
