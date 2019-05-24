package keystore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

func init() {
	// register problems
	problem.RegisterError(sql.ErrNoRows, problem.NotFound)
	problem.RegisterError(errBadKeysBlob, probInvalidInput)

	// register service host as an empty string
	problem.RegisterHost("")
}

func wrapMiddleware(handler http.Handler) http.Handler {
	return recoverHandler(handler)
}

func ServeMux(s *Service) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/store-keys", wrapMiddleware(storeKeysHandler(s.storeKeys)))
	return mux
}

func storeKeysHandler(f interface{}) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			problem.Render(ctx, rw, err)
			return
		}

		var input storeKeysRequest
		err = json.Unmarshal(body, &input)
		if err != nil {
			problem.Render(ctx, rw, err)
		}

		h, err := hal.Handler(f, input)
		if err != nil {
			panic(err)
		}

		h.ServeHTTP(rw, req)
	})
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
