package actions

import (
	"database/sql"
	"net/http"

	"github.com/stellar/go/services/horizon/internal/render"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
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

	initialDataIsFresh bool // Variable that keeps track of whether the data loaded by the setup step is the latest or not
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

		action.SetupAndValidateSSE()
		if base.Err != nil {
			problem.Render(ctx, base.W, base.Err)
			return
		}

		stream := sse.NewStream(ctx, base.W, base.R)

		for {
			action.SSE(stream)

			if base.Err != nil {
				if errors.Cause(base.Err) == sql.ErrNoRows {
					base.Err = errors.New("Object not found")
				} else {
					log.Ctx(ctx).Error(base.Err)
					base.Err = errors.New("Unexpected stream error")
				}
				stream.Err(base.Err)
			}

			if stream.IsDone() {
				return
			}

			select {
			case <-ctx.Done():
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

// Setup runs all setup functions for SSE actions. Setup must be called exactly once.
func (base *Base) Setup(fns ...func()) {
	base.Do(fns...)
	base.initialDataIsFresh = true
}

// NonSetup runs functions that should only be called when the initial data loaded by the setup step is no
// longer fresh. In other words, it's a noop on the first call and only executes functions on subsequent calls.
func (base *Base) NonSetup(fns ...func()) {
	if base.initialDataIsFresh {
		base.initialDataIsFresh = false
		return
	}
	base.Do(fns...)
}
