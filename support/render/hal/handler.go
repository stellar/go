package hal

import (
	"context"
	"net/http"

	"github.com/stellar/go/support/render/problem"
)

// HandlerFunc returns an HTTP HandlerFunc for function fn.
// If fn returns a non-nil error, the handler will use problem.Render.
func HandlerFunc(fn func(context.Context) (interface{}, error)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		res, err := fn(ctx)
		if err != nil {
			problem.Render(ctx, w, err)
		}

		Render(w, res)
	})
}
