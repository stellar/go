package keystore

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

func init() {
	// register errors
	problem.RegisterError(hal.ErrBadRequest, probInvalidRequest)

	// register service host as an empty string
	problem.RegisterHost("")
}

func wrapMiddleware(handler http.Handler) http.Handler {
	return recoverHandler(handler)
}

func ServeMux(s *Service) http.Handler {
	jsonIOHandler := func(f interface{}) http.Handler {
		return wrapMiddleware(jsonHandler(f))
	}

	mux := http.NewServeMux()
	mux.Handle("/store-keys", jsonIOHandler(s.storeKeys))
	return mux
}

func jsonHandler(f interface{}) http.Handler {
	h, err := hal.PostHandler(f)
	if err != nil {
		panic(err)
	}
	return h
}

func recoverHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			if errors.Cause(err) == http.ErrAbortHandler {
				panic(err)
			}

			ctx := req.Context()
			log.Ctx(ctx).WithStack(err).Error(err)
			problem.Render(ctx, rw, err)
		}()

		next.ServeHTTP(rw, req)
	})
}
