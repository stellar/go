package hal

import (
	"context"
	"net/http"

	"github.com/stellar/go/support/render/problem"
)

// Handler returns an HTTP Handler for function fn.
// If fn returns a non-nil error, the handler will use problem.Render.
// TODO: Use reflection to make the hal handler more generic.
func Handler(fn func(context.Context, interface{}) (interface{}, error), params interface{}) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		res, err := fn(ctx, params)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		Render(w, res)
	})
}
