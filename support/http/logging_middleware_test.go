package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stellar/go/support/log"
)

// setXFFMiddleware sets "X-Forwarded-For" header to test LoggingMiddlewareWithOptions.
func setXFFMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		r.Header.Set("X-Forwarded-For", "203.0.113.195")
		next.ServeHTTP(w, r)
	})
}

// setContentMD5MiddleWare sets header to test LoggingMiddlewareWithOptions.
func setContentMD5Middleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		r.Header.Set("Content-MD5", "U3RlbGxhciBpcyBBd2Vzb21lIQ==")
		next.ServeHTTP(w, r)
	})
}

func TestHTTPMiddleware(t *testing.T) {
	mux := chi.NewMux()

	mux.Use(middleware.RequestID)
	mux.Use(LoggingMiddleware)

	mux.Get("/path/{value}", stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		log.Ctx(r.Context()).Info("handler log line")
	}))
	mux.Handle("/not_found", stdhttp.NotFoundHandler())

	src := httptest.NewServer(t, mux)
	src.GET("/path/1234").Expect().Status(stdhttp.StatusOK)
	src.GET("/not_found").Expect().Status(stdhttp.StatusNotFound)
	src.GET("/really_not_found").Expect().Status(stdhttp.StatusNotFound)
}

func TestHTTPMiddlewareWithOptions(t *testing.T) {
	mux := chi.NewMux()

	mux.Use(setXFFMiddleware)
	mux.Use(setContentMD5Middleware)
	mux.Use(middleware.RequestID)
	options := Options{ExtraHeaders: []string{"X-Forwarded-For", "Content-MD5"}}
	mux.Use(LoggingMiddlewareWithOptions(options))

	mux.Get("/path/{value}", stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		log.Ctx(r.Context()).Info("handler log line")
	}))
	mux.Handle("/not_found", stdhttp.NotFoundHandler())

	src := httptest.NewServer(t, mux)
	src.GET("/path/1234").Expect().Status(stdhttp.StatusOK)
	src.GET("/not_found").Expect().Status(stdhttp.StatusNotFound)
	src.GET("/really_not_found").Expect().Status(stdhttp.StatusNotFound)
}

func TestHTTPMiddleware_stdlibServeMux(t *testing.T) {
	mux := stdhttp.ServeMux{}
	mux.Handle(
		"/path/1234",
		middleware.RequestID(
			LoggingMiddleware(
				stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
					log.Ctx(r.Context()).Info("handler log line")
				}),
			),
		),
	)
	mux.Handle(
		"/not_found",
		middleware.RequestID(
			LoggingMiddleware(
				stdhttp.NotFoundHandler(),
			),
		),
	)

	src := httptest.NewServer(t, &mux)
	src.GET("/path/1234").Expect().Status(stdhttp.StatusOK)
	src.GET("/not_found").Expect().Status(stdhttp.StatusNotFound)
	src.GET("/really_not_found").Expect().Status(stdhttp.StatusNotFound)
}

func TestHTTPMiddlewareWithLoggerSet(t *testing.T) {
	logger := log.New()
	mux := chi.NewMux()

	mux.Use(middleware.RequestID)
	mux.Use(SetLoggerMiddleware(logger))
	mux.Use(LoggingMiddleware)

	mux.Get("/path/{value}", stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		log.Ctx(r.Context()).Info("handler log line")
	}))
	mux.Handle("/not_found", stdhttp.NotFoundHandler())

	src := httptest.NewServer(t, mux)
	src.GET("/path/1234").Expect().Status(stdhttp.StatusOK)
	src.GET("/not_found").Expect().Status(stdhttp.StatusNotFound)
	src.GET("/really_not_found").Expect().Status(stdhttp.StatusNotFound)
}
