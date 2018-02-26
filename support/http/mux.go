package http

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/cors"
)

// NewMux returns a new server mux configured with the common defaults used across all
// stellar services.
func NewMux() *chi.Mux {
	mux := chi.NewMux()

	mux.Use(middleware.RequestID)
	mux.Use(middleware.Recoverer)
	mux.Use(LoggingMiddleware)

	return mux
}

// NewAPIMux returns a new server mux configured with the common defaults used for a web API in
// stellar.
func NewAPIMux() *chi.Mux {
	mux := NewMux()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"*"},
	})

	mux.Use(c.Handler)
	return mux
}
