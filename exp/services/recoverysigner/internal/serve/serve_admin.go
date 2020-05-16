package serve

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db"
	"github.com/stellar/go/support/errors"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
)

func serveAdmin(opts Options) {
	deps, err := getAdminHandlerDeps(opts)
	if err != nil {
		opts.Logger.Fatalf("Error: %v", err)
		return
	}

	adminHandler := adminHandler(deps)

	addr := fmt.Sprintf(":%d", opts.AdminPort)
	supporthttp.Run(supporthttp.Config{
		ListenAddr: addr,
		Handler:    adminHandler,
		OnStarting: func() {
			deps.Logger.Infof("Starting admin port server on %s", addr)
		},
	})
}

type adminHandlerDeps struct {
	Logger           *supportlog.Entry
	AccountStore     account.Store
	MetricsNamespace string
}

func getAdminHandlerDeps(opts Options) (adminHandlerDeps, error) {
	db, err := db.Open(opts.DatabaseURL)
	if err != nil {
		return adminHandlerDeps{}, errors.Wrap(err, "error parsing database url")
	}
	err = db.Ping()
	if err != nil {
		opts.Logger.Warn("Error pinging to Database: ", err)
	}
	accountStore := &account.DBStore{DB: db}

	deps := adminHandlerDeps{
		Logger:           opts.Logger,
		AccountStore:     accountStore,
		MetricsNamespace: opts.MetricsNamespace,
	}
	return deps, nil
}

func adminHandler(deps adminHandlerDeps) http.Handler {
	mux := supporthttp.NewMux(deps.Logger)
	mux.Handle("/metrics", promhttp.HandlerFor(metricsHandler{
		Logger:       deps.Logger,
		AccountStore: deps.AccountStore,
		Namespace:    deps.MetricsNamespace,
	}.Registry(), promhttp.HandlerOpts{}))
	return mux
}
