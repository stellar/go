package hal

import (
	"context"
	"net/http"

	"github.com/stellar/go/support/render/problem"
)

// handler is an http.Handler that calls a function for each request.
type handler struct {
	fn func(context.Context) (interface{}, error)
}

// Handler returns an HTTP handler for function fn.
// If fn returns a non-nil error, the handler will use problem.Render.
func Handler(fn func(context.Context) (interface{}, error)) http.Handler {
	return &handler{fn}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	res, err := h.fn(ctx)
	if err != nil {
		problem.Render(ctx, w, err)
	}

	Render(w, res)
}
