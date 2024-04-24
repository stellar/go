package ledgerexporter

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stellar/go/ingest/ledgerbackend"
	_ "github.com/stellar/go/network"
	supporthttp "github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
)

var (
	logger = log.New().WithField("service", "ledger-exporter")
)

type App struct {
	config             Config
	ledgerBackend      ledgerbackend.LedgerBackend
	dataStore          DataStore
	exportManager      *ExportManager
	uploader           Uploader
	prometheusRegistry *prometheus.Registry
}

func NewApp() *App {
	logger.SetLevel(log.DebugLevel)

	config := Config{}
	err := config.LoadConfig()
	logFatalIf(err, "Could not load configuration")

	app := &App{config: config, prometheusRegistry: prometheus.NewRegistry()}
	app.prometheusRegistry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{Namespace: "ledger_exporter"}),
		collectors.NewGoCollector(),
	)
	return app
}

func (a *App) init(ctx context.Context) {
	a.dataStore = mustNewDataStore(ctx, a.config)
	a.ledgerBackend = mustNewLedgerBackend(ctx, a.config, a.prometheusRegistry)
	// TODO: make number of upload workers configurable instead of hard coding it to 1
	queue := NewUploadQueue(1, a.prometheusRegistry)
	a.exportManager = NewExportManager(a.config.ExporterConfig, a.ledgerBackend, queue, a.prometheusRegistry)
	a.uploader = NewUploader(
		a.dataStore,
		queue,
		a.prometheusRegistry,
	)
}

func (a *App) close() {
	if err := a.dataStore.Close(); err != nil {
		logger.WithError(err).Error("Error closing datastore")
	}
	if err := a.ledgerBackend.Close(); err != nil {
		logger.WithError(err).Error("Error closing ledgerBackend")
	}
}

func (a *App) serveAdmin() {
	if a.config.AdminPort == 0 {
		return
	}

	mux := supporthttp.NewMux(logger)
	mux.Handle("/metrics", promhttp.HandlerFor(a.prometheusRegistry, promhttp.HandlerOpts{}))

	addr := fmt.Sprintf(":%d", a.config.AdminPort)
	supporthttp.Run(supporthttp.Config{
		ListenAddr: addr,
		Handler:    mux,
		OnStarting: func() {
			logger.Infof("Starting admin port server on %s", addr)
		},
	})
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.init(ctx)
	defer a.close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := a.uploader.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.WithError(err).Error("Error executing Uploader")
			cancel()
		}
	}()

	go func() {
		defer wg.Done()

		err := a.exportManager.Run(ctx, a.config.StartLedger, a.config.EndLedger)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.WithError(err).Error("Error executing ExportManager")
			cancel()
		}
	}()

	go a.serveAdmin()

	// Handle OS signals to gracefully terminate the service
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Infof("Received termination signal: %v", sig)
		cancel()
	}()

	wg.Wait()
	logger.Info("Shutting down ledger-exporter")
}

func mustNewDataStore(ctx context.Context, config Config) DataStore {
	dataStore, err := NewDataStore(ctx, fmt.Sprintf("%s/%s", config.DestinationURL, config.Network))
	logFatalIf(err, "Could not connect to destination data store")
	return dataStore
}

// mustNewLedgerBackend Creates and initializes captive core ledger backend
// Currently, only supports captive-core as ledger backend
func mustNewLedgerBackend(ctx context.Context, config Config, prometheusRegistry *prometheus.Registry) ledgerbackend.LedgerBackend {
	captiveConfig := config.GenerateCaptiveCoreConfig()

	var backend ledgerbackend.LedgerBackend
	var err error
	// Create a new captive core backend
	backend, err = ledgerbackend.NewCaptive(captiveConfig)
	logFatalIf(err, "Failed to create captive-core instance")
	backend = ledgerbackend.WithMetrics(backend, prometheusRegistry, "ledger_exporter")

	var ledgerRange ledgerbackend.Range
	if config.EndLedger == 0 {
		ledgerRange = ledgerbackend.UnboundedRange(config.StartLedger)
	} else {
		ledgerRange = ledgerbackend.BoundedRange(config.StartLedger, config.EndLedger)
	}

	err = backend.PrepareRange(ctx, ledgerRange)
	logFatalIf(err, "Could not prepare captive core ledger backend")
	return backend
}

func logFatalIf(err error, message string, args ...interface{}) {
	if err != nil {
		logger.WithError(err).Fatalf(message, args...)
	}
}
