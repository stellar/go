package keystore

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

var errBadKeysBlob = errors.New("invalid base64 encoding string")

func init() {
	// register problems
	problem.RegisterError(sql.ErrNoRows, problem.NotFound)

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

func storeKeysHandler(f func(context.Context, []byte) ([]byte, error)) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			problem.Render(ctx, rw, err)
			return
		}

		data, err := base64.RawURLEncoding.DecodeString(string(body))
		if err != nil {
			problem.Render(ctx, rw, errBadKeysBlob)
			return
		}

		encodeResult := func(context.Context) (string, error) {
			res, err := f(ctx, data)
			if err != nil {
				return "", err
			}

			return base64.RawURLEncoding.EncodeToString(res), nil
		}

		h, err := hal.Handler(encodeResult, nil)
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
