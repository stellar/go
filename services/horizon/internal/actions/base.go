package actions

import (
	"net/http"

	"github.com/stellar/go/services/horizon/internal/render"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/support/render/problem"
)

// Base is a helper struct you can use as part of a custom action via
// composition.
//
// TODO: example usage
type Base struct {
	W   http.ResponseWriter
	R   *http.Request
	Err error

	isSetup bool
}

// Prepare established the common attributes that get used in nearly every
// action.  "Child" actions may override this method to extend action, but it
// is advised you also call this implementation to maintain behavior.
func (base *Base) Prepare(w http.ResponseWriter, r *http.Request) {
	base.W = w
	base.R = r
}

// Execute trigger content negotiation and the actual execution of one of the
// action's handlers.
func (base *Base) Execute(action interface{}) {
	ctx := base.R.Context()
	contentType := render.Negotiate(base.R)

	switch contentType {
	case render.MimeHal, render.MimeJSON:
		action, ok := action.(JSON)

		if !ok {
			goto NotAcceptable
		}

		action.JSON()

		if base.Err != nil {
			problem.Render(ctx, base.W, base.Err)
			return
		}

	case render.MimeEventStream:
		action, ok := action.(SSE)
		if !ok {
			goto NotAcceptable
		}

		stream := sse.NewStream(ctx, base.W, base.R)
		// If WritePreamble fails, it means flush isn't supported by base.W so return immediately.
		if stream.IsDone() {
			return
		}

		for {
			action.SSE(stream)

			if base.Err != nil {
				stream.Err(base.Err)
			}

			if stream.IsDone() {
				return
			}

			select {
			case <-ctx.Done():
					stream.Done() // Call Done on the stream so that it doesn't send any more heartbeats.
					return
			case <-sse.Pumped():
				//no-op, continue onto the next iteration
			}
		}
	case render.MimeRaw:
		action, ok := action.(Raw)

		if !ok {
			goto NotAcceptable
		}

		action.Raw()

		if base.Err != nil {
			problem.Render(ctx, base.W, base.Err)
			return
		}
	default:
		goto NotAcceptable
	}
	return

NotAcceptable:
	problem.Render(ctx, base.W, hProblem.NotAcceptable)
	return
}

// Do executes the provided func iff there is no current error for the action.
// Provides a nicer way to invoke a set of steps that each may set `action.Err`
// during execution
func (base *Base) Do(fns ...func()) {
	for _, fn := range fns {
		if base.Err != nil {
			return
		}

		fn()
	}
}

// Setup runs the provided funcs if and only if no call to Setup() has been
// made previously on this action.
func (base *Base) Setup(fns ...func()) {
	if base.isSetup {
		return
	}
	base.Do(fns...)
	base.isSetup = true
}
