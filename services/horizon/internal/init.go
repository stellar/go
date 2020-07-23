package horizon

import (
	"context"
	"net/http"

	"github.com/getsentry/raven-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/services/horizon/internal/txsub/sequence"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
)

func mustNewDBSession(databaseURL string, maxIdle, maxOpen int) *db.Session {
	session, err := db.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("cannot open Horizon DB: %v", err)
	}

	session.DB.SetMaxIdleConns(maxIdle)
	session.DB.SetMaxOpenConns(maxOpen)
	return session
}

func mustInitHorizonDB(app *App) {
	maxIdle := app.config.HorizonDBMaxIdleConnections
	maxOpen := app.config.HorizonDBMaxOpenConnections
	if app.config.Ingest {
		maxIdle -= expingest.MaxDBConnections
		maxOpen -= expingest.MaxDBConnections
		if maxIdle <= 0 {
			log.Fatalf("max idle connections to horizon db must be greater than %d", expingest.MaxDBConnections)
		}
		if maxOpen <= 0 {
			log.Fatalf("max open connections to horizon db must be greater than %d", expingest.MaxDBConnections)
		}
	}

	app.historyQ = &history.Q{mustNewDBSession(
		app.config.DatabaseURL,
		maxIdle,
		maxOpen,
	)}
}

func initExpIngester(app *App) {
	var err error
	app.expingester, err = expingest.NewSystem(expingest.Config{
		CoreSession: mustNewDBSession(
			app.config.StellarCoreDatabaseURL, expingest.MaxDBConnections, expingest.MaxDBConnections,
		),
		HistorySession: mustNewDBSession(
			app.config.DatabaseURL, expingest.MaxDBConnections, expingest.MaxDBConnections,
		),
		NetworkPassphrase: app.config.NetworkPassphrase,
		// TODO:
		// Use the first archive for now. We don't have a mechanism to
		// use multiple archives at the same time currently.
		HistoryArchiveURL:        app.config.HistoryArchiveURLs[0],
		StellarCoreURL:           app.config.StellarCoreURL,
		StellarCoreCursor:        app.config.CursorName,
		StellarCoreBinaryPath:    app.config.StellarCoreBinaryPath,
		StellarCoreConfigPath:    app.config.StellarCoreConfigPath,
		DisableStateVerification: app.config.IngestDisableStateVerification,
	})

	if err != nil {
		log.Fatal(err)
	}
}

func initPathFinder(app *App) {
	orderBookGraph := orderbook.NewOrderBookGraph()
	app.orderBookStream = expingest.NewOrderBookStream(
		&history.Q{app.HorizonSession(app.ctx)},
		orderBookGraph,
	)

	app.paths = simplepath.NewInMemoryFinder(orderBookGraph)
}

// initSentry initialized the default sentry client with the configured DSN
func initSentry(app *App) {
	if app.config.SentryDSN == "" {
		return
	}

	log.WithField("dsn", app.config.SentryDSN).Info("Initializing sentry")
	err := raven.SetDSN(app.config.SentryDSN)
	if err != nil {
		log.Fatal(err)
	}
}

// initLogglyLog attaches a loggly hook to our logging system.
func initLogglyLog(app *App) {
	if app.config.LogglyToken == "" {
		return
	}

	log.WithFields(log.F{
		"token": app.config.LogglyToken,
		"tag":   app.config.LogglyTag,
	}).Info("Initializing loggly hook")

	hook := log.NewLogglyHook(app.config.LogglyToken, app.config.LogglyTag)
	log.DefaultLogger.Logger.Hooks.Add(hook)

	go func() {
		<-app.ctx.Done()
		hook.Flush()
	}()
}

func initDbMetrics(app *App) {
	app.historyLatestLedgerGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "history", Name: "latest_ledger",
	})
	app.prometheusRegistry.MustRegister(app.historyLatestLedgerGauge)

	app.historyElderLedgerGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "history", Name: "elder_ledger",
	})
	app.prometheusRegistry.MustRegister(app.historyElderLedgerGauge)

	app.coreLatestLedgerGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "stellar_core", Name: "latest_ledger",
	})
	app.prometheusRegistry.MustRegister(app.coreLatestLedgerGauge)

	app.dbOpenConnectionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "db", Name: "open_connections",
	})
	app.prometheusRegistry.MustRegister(app.dbOpenConnectionsGauge)

	app.dbInUseConnectionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "db", Name: "in_use_connections",
	})
	app.prometheusRegistry.MustRegister(app.dbInUseConnectionsGauge)

	app.dbWaitCountGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "db", Name: "wait_count",
	})
	app.prometheusRegistry.MustRegister(app.dbWaitCountGauge)

	app.dbWaitDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "horizon", Subsystem: "db", Name: "wait_duration",
	})
	app.prometheusRegistry.MustRegister(app.dbWaitDurationGauge)

	app.prometheusRegistry.MustRegister(app.orderBookStream.LatestLedgerGauge)
}

// initIngestMetrics registers the metrics for the ingestion into the provided
// app's metrics registry.
func initIngestMetrics(app *App) {
	if app.expingester == nil {
		return
	}

	app.prometheusRegistry.MustRegister(app.expingester.Metrics().LedgerIngestionDuration)
	app.prometheusRegistry.MustRegister(app.expingester.Metrics().StateVerifyDuration)
}

func initTxSubMetrics(app *App) {
	app.submitter.Init()
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.SubmissionDuration)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.BufferedSubmissionsGauge)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.OpenSubmissionsGauge)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.FailedSubmissionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.SuccessfulSubmissionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.V0TransactionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.V1TransactionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.FeeBumpTransactionsCounter)
}

func initWebMetrics(app *App) {
	app.prometheusRegistry.MustRegister(app.web.requestDuration)
}

func initSubmissionSystem(app *App) {
	app.submitter = &txsub.System{
		Pending:         txsub.NewDefaultSubmissionList(),
		Submitter:       txsub.NewDefaultSubmitter(http.DefaultClient, app.config.StellarCoreURL),
		SubmissionQueue: sequence.NewManager(),
		DB: func(ctx context.Context) txsub.HorizonDB {
			return &history.Q{Session: app.HorizonSession(ctx)}
		},
	}
}
