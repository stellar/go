package http

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/cors"
	"github.com/stellar/go/support/log"
)

// NewMux returns a new server mux configured with the common defaults used across all
// stellar services.
func NewMux(options ...MuxOption) *chi.Mux {
	muxOptions := NewMuxOptions(options...)

	mux := chi.NewMux()

	mux.Use(middleware.RequestID)
	mux.Use(middleware.Recoverer)
	if muxOptions.Logger != nil {
		mux.Use(SetLoggerMiddleware(muxOptions.Logger))
	}
	mux.Use(LoggingMiddleware)

	return mux
}

// NewAPIMux returns a new server mux configured with the common defaults used for a web API in
// stellar.
func NewAPIMux(options ...MuxOption) *chi.Mux {
	mux := NewMux(options...)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"GET", "PUT", "POST", "PATCH", "DELETE", "HEAD", "OPTIONS"},
	})

	mux.Use(c.Handler)
	return mux
}

// MuxOptions is a set of options that can be optionally set when calling
// NewMux or NewAPIMux.
type MuxOptions struct {
	Logger *log.Entry
}

// NewMuxOptions creates a MuxOptions from a set of MuxOption.
func NewMuxOptions(options ...MuxOption) MuxOptions {
	mo := MuxOptions{}
	for _, o := range options {
		mo = o(mo)
	}
	return mo
}

// MuxOption is a function that sets mux options.
type MuxOption func(options MuxOptions) MuxOptions

// WithMuxOptionLogger sets a logger when instantiating a mux.
func WithMuxOptionLogger(l *log.Entry) func(MuxOptions) MuxOptions {
	return func(options MuxOptions) MuxOptions {
		options.Logger = l
		return options
	}
}
