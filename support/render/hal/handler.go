package hal

import (
	"context"
	"net/http"

	"github.com/stellar/go/support/render/problem"
)

// Handler returns an HTTP Handler for function fn.
// If fn returns a non-nil error, the handler will use problem.Render.
func Handler(fn func(context.Context) (interface{}, error)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		res, err := fn(ctx)
		if err != nil {
			problem.Render(ctx, w, err)
		}

		Render(w, res)
	})
}
