package keystore

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
)

func init() {
	// register errors
	problem.RegisterError(httpjson.ErrBadRequest, probInvalidRequest)
	problem.RegisterError(sql.ErrNoRows, problem.NotFound)

	// register service host as an empty string
	problem.RegisterHost("")
}

func wrapMiddleware(handler http.Handler) http.Handler {
	return recoverHandler(handler)
}

func ServeMux(s *Service) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/keys", wrapMiddleware(s.keysHTTPMethodHandler()))
	return mux
}

func (s *Service) keysHTTPMethodHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			jsonHandler(s.getKeys).ServeHTTP(rw, req)

		case http.MethodPut:
			jsonHandler(s.putKeys).ServeHTTP(rw, req)

		case http.MethodDelete:
			jsonHandler(s.deleteKeys).ServeHTTP(rw, req)

		default:
			problem.Render(req.Context(), rw, probMethodNotAllowed)
		}
	})
}

func jsonHandler(f interface{}) http.Handler {
	h, err := httpjson.ReqBodyHandler(f, httpjson.JSON)
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
