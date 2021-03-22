package horizon

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
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
		maxIdle -= ingest.MaxDBConnections
		maxOpen -= ingest.MaxDBConnections
		if maxIdle <= 0 {
			log.Fatalf("max idle connections to horizon db must be greater than %d", ingest.MaxDBConnections)
		}
		if maxOpen <= 0 {
			log.Fatalf("max open connections to horizon db must be greater than %d", ingest.MaxDBConnections)
		}
	}

	app.historyQ = &history.Q{mustNewDBSession(
		app.config.DatabaseURL,
		maxIdle,
		maxOpen,
	)}
}

func initIngester(app *App) {
	var err error
	var coreSession *db.Session
	if !app.config.EnableCaptiveCoreIngestion {
		coreSession = mustNewDBSession(app.config.StellarCoreDatabaseURL, ingest.MaxDBConnections, ingest.MaxDBConnections)
	}
	app.ingester, err = ingest.NewSystem(ingest.Config{
		CoreSession: coreSession,
		HistorySession: mustNewDBSession(
			app.config.DatabaseURL, ingest.MaxDBConnections, ingest.MaxDBConnections,
		),
		NetworkPassphrase: app.config.NetworkPassphrase,
		// TODO:
		// Use the first archive for now. We don't have a mechanism to
		// use multiple archives at the same time currently.
		HistoryArchiveURL:           app.config.HistoryArchiveURLs[0],
		CheckpointFrequency:         app.config.CheckpointFrequency,
		StellarCoreURL:              app.config.StellarCoreURL,
		StellarCoreCursor:           app.config.CursorName,
		CaptiveCoreBinaryPath:       app.config.CaptiveCoreBinaryPath,
		CaptiveCoreConfigAppendPath: app.config.CaptiveCoreConfigAppendPath,
		CaptiveCoreHTTPPort:         app.config.CaptiveCoreHTTPPort,
		CaptiveCoreLogPath:          app.config.CaptiveCoreLogPath,
		RemoteCaptiveCoreURL:        app.config.RemoteCaptiveCoreURL,
		EnableCaptiveCore:           app.config.EnableCaptiveCoreIngestion,
		DisableStateVerification:    app.config.IngestDisableStateVerification,
	})

	if err != nil {
		log.Fatal(err)
	}
}

func initPathFinder(app *App) {
	orderBookGraph := orderbook.NewOrderBookGraph()
	app.orderBookStream = ingest.NewOrderBookStream(
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
	app.buildInfoGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "horizon", Subsystem: "build", Name: "info"},
		[]string{"version", "goversion"},
	)
	app.prometheusRegistry.MustRegister(app.buildInfoGauge)
	app.buildInfoGauge.With(prometheus.Labels{
		"version":   app.horizonVersion,
		"goversion": runtime.Version(),
	}).Inc()

	app.ingestingGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{Namespace: "horizon", Subsystem: "ingest", Name: "enabled"},
	)
	app.prometheusRegistry.MustRegister(app.ingestingGauge)

	app.historyLatestLedgerCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{Namespace: "horizon", Subsystem: "history", Name: "latest_ledger"},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return float64(ls.HistoryLatest)
		},
	)
	app.prometheusRegistry.MustRegister(app.historyLatestLedgerCounter)

	app.historyLatestLedgerClosedAgoGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "history", Name: "latest_ledger_closed_ago_seconds",
			Help: "seconds since the close of the last ingested ledger",
		},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return time.Since(ls.HistoryLatestClosedAt).Seconds()
		},
	)
	app.prometheusRegistry.MustRegister(app.historyLatestLedgerClosedAgoGauge)

	app.historyElderLedgerCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{Namespace: "horizon", Subsystem: "history", Name: "elder_ledger"},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return float64(ls.HistoryElder)
		},
	)
	app.prometheusRegistry.MustRegister(app.historyElderLedgerCounter)

	app.coreLatestLedgerCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{Namespace: "horizon", Subsystem: "stellar_core", Name: "latest_ledger"},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return float64(ls.CoreLatest)
		},
	)
	app.prometheusRegistry.MustRegister(app.coreLatestLedgerCounter)

	app.coreSynced = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "stellar_core", Name: "synced",
			Help: "determines if Stellar-Core defined by --stellar-core-url is synced with the network",
		},
		func() float64 {
			if app.coreSettings.Synced {
				return 1
			} else {
				return 0
			}
		},
	)
	app.prometheusRegistry.MustRegister(app.coreSynced)

	app.dbMaxOpenConnectionsGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Namespace: "horizon", Subsystem: "db", Name: "max_open_connections"},
		func() float64 {
			// Right now MaxOpenConnections in Horizon is static however it's possible that
			// it will change one day. In such case, using GaugeFunc is very cheap and will
			// prevent issues with this metric in the future.
			return float64(app.historyQ.Session.DB.Stats().MaxOpenConnections)
		},
	)
	app.prometheusRegistry.MustRegister(app.dbMaxOpenConnectionsGauge)

	app.dbOpenConnectionsGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Namespace: "horizon", Subsystem: "db", Name: "open_connections"},
		func() float64 {
			return float64(app.historyQ.Session.DB.Stats().OpenConnections)
		},
	)
	app.prometheusRegistry.MustRegister(app.dbOpenConnectionsGauge)

	app.dbInUseConnectionsGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{Namespace: "horizon", Subsystem: "db", Name: "in_use_connections"},
		func() float64 {
			return float64(app.historyQ.Session.DB.Stats().InUse)
		},
	)
	app.prometheusRegistry.MustRegister(app.dbInUseConnectionsGauge)

	app.dbWaitCountCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "db", Name: "wait_count_total",
			Help: "total number of number of connections waited for",
		},
		func() float64 {
			return float64(app.historyQ.Session.DB.Stats().WaitCount)
		},
	)
	app.prometheusRegistry.MustRegister(app.dbWaitCountCounter)

	app.dbWaitDurationCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Namespace: "horizon", Subsystem: "db", Name: "wait_duration_seconds_total",
			Help: "total time blocked waiting for a new connection",
		},
		func() float64 {
			return app.historyQ.Session.DB.Stats().WaitDuration.Seconds()
		},
	)
	app.prometheusRegistry.MustRegister(app.dbWaitDurationCounter)

	app.prometheusRegistry.MustRegister(app.orderBookStream.LatestLedgerGauge)
}

// initGoMetrics registers the Go collector provided by prometheus package which
// includes Go-related metrics.
func initGoMetrics(app *App) {
	app.prometheusRegistry.MustRegister(prometheus.NewGoCollector())
}

// initProcessMetrics registers the process collector provided by prometheus
// package. This is only available on operating systems with a Linux-style proc
// filesystem and on Microsoft Windows.
func initProcessMetrics(app *App) {
	app.prometheusRegistry.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
}

// initIngestMetrics registers the metrics for the ingestion into the provided
// app's metrics registry.
func initIngestMetrics(app *App) {
	if app.ingester == nil {
		return
	}

	app.ingestingGauge.Inc()
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LocalLatestLedger)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LedgerIngestionDuration)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().StateVerifyDuration)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().StateInvalidGauge)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LedgerStatsCounter)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().ProcessorsRunDuration)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().CaptiveStellarCoreSynced)
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
	app.prometheusRegistry.MustRegister(app.webServer.Metrics.RequestDurationSummary)
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
