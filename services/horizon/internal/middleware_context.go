package horizon

import (
	"context"
	"net/http"

	gctx "github.com/goji/context"
	"github.com/stellar/go/services/horizon/internal/context/requestid"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/zenazn/goji/web"
)

func contextMiddleware(parent context.Context) func(c *web.C, next http.Handler) http.Handler {
	return func(c *web.C, next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := parent
			ctx = requestid.ContextFromC(ctx, c)
			ctx, cancel := httpx.RequestContext(ctx, w, r)

			gctx.Set(c, ctx)
			defer cancel()
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
