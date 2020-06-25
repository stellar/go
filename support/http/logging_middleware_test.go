package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
)

func TestHTTPMiddleware(t *testing.T) {
	done := log.DefaultLogger.StartTest(log.InfoLevel)
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

	// get the log buffer and ensure it has both the start and end log lines for
	// each request
	logged := done()
	if assert.Len(t, logged, 7, "unexpected log line count") {
		assert.Equal(t, "starting request", logged[0].Message)
		assert.Equal(t, "http", logged[0].Data["subsys"])
		assert.Equal(t, "GET", logged[0].Data["method"])
		assert.NotEmpty(t, logged[0].Data["req"])
		assert.Equal(t, "/path/1234", logged[0].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[0].Data["useragent"])
		req1 := logged[0].Data["req"]

		assert.Equal(t, "handler log line", logged[1].Message)
		assert.Equal(t, req1, logged[1].Data["req"])

		assert.Equal(t, "finished request", logged[2].Message)
		assert.Equal(t, "http", logged[2].Data["subsys"])
		assert.Equal(t, "GET", logged[2].Data["method"])
		assert.Equal(t, req1, logged[2].Data["req"])
		assert.Equal(t, "/path/1234", logged[2].Data["path"])
		assert.Equal(t, "/path/{value}", logged[2].Data["route"])

		assert.Equal(t, "starting request", logged[3].Message)
		assert.Equal(t, "http", logged[3].Data["subsys"])
		assert.Equal(t, "GET", logged[3].Data["method"])
		assert.NotEmpty(t, logged[3].Data["req"])
		assert.NotEmpty(t, logged[3].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[3].Data["useragent"])
		req2 := logged[3].Data["req"]

		assert.Equal(t, "finished request", logged[4].Message)
		assert.Equal(t, "http", logged[4].Data["subsys"])
		assert.Equal(t, "GET", logged[4].Data["method"])
		assert.Equal(t, req2, logged[4].Data["req"])
		assert.Equal(t, "/not_found", logged[4].Data["path"])
		assert.Equal(t, "/not_found", logged[4].Data["route"])

		assert.Equal(t, "starting request", logged[5].Message)
		assert.Equal(t, "http", logged[5].Data["subsys"])
		assert.Equal(t, "GET", logged[5].Data["method"])
		assert.NotEmpty(t, logged[5].Data["req"])
		assert.NotEmpty(t, logged[5].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[5].Data["useragent"])
		req3 := logged[5].Data["req"]

		assert.Equal(t, "finished request", logged[6].Message)
		assert.Equal(t, "http", logged[6].Data["subsys"])
		assert.Equal(t, "GET", logged[6].Data["method"])
		assert.Equal(t, req3, logged[6].Data["req"])
		assert.Equal(t, "/really_not_found", logged[6].Data["path"])
		assert.Equal(t, "", logged[6].Data["route"])
	}
}

func TestHTTPMiddleware_stdlibServeMux(t *testing.T) {
	done := log.DefaultLogger.StartTest(log.InfoLevel)

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

	// get the log buffer and ensure it has both the start and end log lines for
	// each request
	logged := done()
	if assert.Len(t, logged, 5, "unexpected log line count") {
		assert.Equal(t, "starting request", logged[0].Message)
		assert.Equal(t, "http", logged[0].Data["subsys"])
		assert.Equal(t, "GET", logged[0].Data["method"])
		assert.NotEmpty(t, logged[0].Data["req"])
		assert.Equal(t, "/path/1234", logged[0].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[0].Data["useragent"])
		req1 := logged[0].Data["req"]

		assert.Equal(t, "handler log line", logged[1].Message)
		assert.Equal(t, req1, logged[1].Data["req"])

		assert.Equal(t, "finished request", logged[2].Message)
		assert.Equal(t, "http", logged[2].Data["subsys"])
		assert.Equal(t, "GET", logged[2].Data["method"])
		assert.Equal(t, req1, logged[2].Data["req"])
		assert.Equal(t, "/path/1234", logged[2].Data["path"])
		assert.Equal(t, nil, logged[2].Data["route"])

		assert.Equal(t, "starting request", logged[3].Message)
		assert.Equal(t, "http", logged[3].Data["subsys"])
		assert.Equal(t, "GET", logged[3].Data["method"])
		assert.NotEmpty(t, logged[3].Data["req"])
		assert.NotEmpty(t, logged[3].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[3].Data["useragent"])
		req2 := logged[3].Data["req"]

		assert.Equal(t, "finished request", logged[4].Message)
		assert.Equal(t, "http", logged[4].Data["subsys"])
		assert.Equal(t, "GET", logged[4].Data["method"])
		assert.Equal(t, req2, logged[4].Data["req"])
		assert.Equal(t, "/not_found", logged[4].Data["path"])
		assert.Equal(t, nil, logged[4].Data["route"])
	}
}

func TestHTTPMiddlewareWithLoggerSet(t *testing.T) {
	logger := log.New()
	done := logger.StartTest(log.InfoLevel)
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

	// get the log buffer and ensure it has both the start and end log lines for
	// each request
	logged := done()
	if assert.Len(t, logged, 7, "unexpected log line count") {
		assert.Equal(t, "starting request", logged[0].Message)
		assert.Equal(t, "http", logged[0].Data["subsys"])
		assert.Equal(t, "GET", logged[0].Data["method"])
		assert.NotEmpty(t, logged[0].Data["req"])
		assert.Equal(t, "/path/1234", logged[0].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[0].Data["useragent"])
		req1 := logged[0].Data["req"]

		assert.Equal(t, "handler log line", logged[1].Message)
		assert.Equal(t, req1, logged[1].Data["req"])

		assert.Equal(t, "finished request", logged[2].Message)
		assert.Equal(t, "http", logged[2].Data["subsys"])
		assert.Equal(t, "GET", logged[2].Data["method"])
		assert.Equal(t, req1, logged[2].Data["req"])
		assert.Equal(t, "/path/1234", logged[2].Data["path"])
		assert.Equal(t, "/path/{value}", logged[2].Data["route"])

		assert.Equal(t, "starting request", logged[3].Message)
		assert.Equal(t, "http", logged[3].Data["subsys"])
		assert.Equal(t, "GET", logged[3].Data["method"])
		assert.NotEmpty(t, logged[3].Data["req"])
		assert.NotEmpty(t, logged[3].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[3].Data["useragent"])
		req2 := logged[3].Data["req"]

		assert.Equal(t, "finished request", logged[4].Message)
		assert.Equal(t, "http", logged[4].Data["subsys"])
		assert.Equal(t, "GET", logged[4].Data["method"])
		assert.Equal(t, req2, logged[4].Data["req"])
		assert.Equal(t, "/not_found", logged[4].Data["path"])
		assert.Equal(t, "/not_found", logged[4].Data["route"])

		assert.Equal(t, "starting request", logged[5].Message)
		assert.Equal(t, "http", logged[5].Data["subsys"])
		assert.Equal(t, "GET", logged[5].Data["method"])
		assert.NotEmpty(t, logged[5].Data["req"])
		assert.NotEmpty(t, logged[5].Data["path"])
		assert.Equal(t, "Go-http-client/1.1", logged[5].Data["useragent"])
		req3 := logged[5].Data["req"]

		assert.Equal(t, "finished request", logged[6].Message)
		assert.Equal(t, "http", logged[6].Data["subsys"])
		assert.Equal(t, "GET", logged[6].Data["method"])
		assert.Equal(t, req3, logged[6].Data["req"])
		assert.Equal(t, "/really_not_found", logged[6].Data["path"])
		assert.Equal(t, "", logged[6].Data["route"])
	}
}
