package federation

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/rs/cors"
	sdb "github.com/stellar/go/internal/db"
	"goji.io"
	"goji.io/pat"
)

// NewApp constructs an new App instance from the provided config.
func NewApp(config Config) (*App, error) {
	var dialect string

	switch config.Database.Type {
	case "mysql":
		dialect = "mysql"
	case "postgres":
		dialect = "postgres"
	case "sqlite3":
		dialect = "sqlite3"
	default:
		return nil, errors.Errorf("Invalid db type: %s", config.Database.Type)
	}

	repo, err := sdb.Open(dialect, config.Database.URL)
	if err != nil {
		return nil, errors.Wrap(err, "db open failed")
	}

	return &App{config: config, db: repo}, nil
}

// Handler builds a new http.Handler that will handle federation requests using
// the config and database of `a`.
func (a *App) Handler() http.Handler {
	mux := goji.NewMux()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{"GET"},
	})
	mux.Use(c.Handler)
	mux.Use(jsonMiddleware)

	requestHandler := &RequestHandler{config: &a.config, db: a.db}
	mux.Handle(pat.Get("/federation"), requestHandler)
	mux.Handle(pat.Get("/federation/"), requestHandler)

	return mux
}
