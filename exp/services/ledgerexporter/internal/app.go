package exporter

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/stellar/go/ingest/ledgerbackend"
	_ "github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
)

var (
	logger = supportlog.New().WithField("service", "ledger-exporter")
)

type App struct {
	config             Config
	backend            ledgerbackend.LedgerBackend
	destinationStorage DataStore
	exportManager      ExportManager
	uploader           Uploader
}

func NewApp() *App {
	logger.SetLevel(supportlog.DebugLevel)

	config := Config{}
	err := config.LoadConfig()
	logFatalIf(err, "Could not load configuration")

	app := &App{config: config}
	return app
}

func (a *App) init(ctx context.Context) {
	a.destinationStorage = mustNewDataStore(ctx, &a.config)
	a.backend = mustNewLedgerBackend(ctx, a.config)
	a.exportManager = NewExportManager(a.config.ExporterConfig, a.backend)
	a.uploader = NewUploader(a.destinationStorage, a.exportManager.GetMetaArchiveChannel())
}

func (a *App) close() {
	//TODO: error handling
	a.destinationStorage.Close()
	a.backend.Close()
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.init(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := a.uploader.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Errorf("Error executing Uploader: %v", err)
			cancel()
		}
	}()

	go func() {
		defer wg.Done()

		err := a.exportManager.Run(ctx, a.config.StartLedger, a.config.EndLedger)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Errorf("Error executing ExportManager: %v", err)
			cancel()
		}
	}()

	go func() {
		// Handle OS signals to gracefully terminate the service
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-ctx.Done():
				logger.Infof("Received context done signal")
				return
			case sig := <-sigCh:
				logger.Infof("Received signal: %v", sig)
				cancel()
				return
			}
		}
	}()

	wg.Wait()

	a.close()

	logger.Info("Shutting down ledger-exporter.")
}

func mustNewDataStore(ctx context.Context, config *Config) DataStore {
	destinationStorage, err := NewDataStore(ctx, fmt.Sprintf("%s/%s", config.DestinationURL, config.Network))
	logFatalIf(err, "Could not connect to destination storage")
	return destinationStorage
}

// mustNewLedgerBackend Creates and initializes captive core ledger backend
// Currently, only supports captive-core as ledger backend
func mustNewLedgerBackend(ctx context.Context, config Config) ledgerbackend.LedgerBackend {
	captiveConfig := config.GenerateCaptiveCoreConfig()

	// Create a new captive core backend
	backend, err := ledgerbackend.NewCaptive(captiveConfig)
	logFatalIf(err, "Failed to create captive-core instance")

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
