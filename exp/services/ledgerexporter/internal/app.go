package exporter

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/stellar/go/ingest/ledgerbackend"
	_ "github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
)

var (
	logger = supportlog.New().WithField("service", "ledger-exporter")
)

type App struct {
	ctx                context.Context
	cancel             func()
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
	app.init()
	return app
}

func (a *App) init() {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.destinationStorage = NewDestinationStorage(a.ctx, &a.config)
	a.backend = NewLedgerBackend(a.ctx, a.config)
	a.exportManager = NewExportManager(a.config.ExporterConfig, a.backend)
	a.uploader = NewUploader(a.destinationStorage, a.exportManager.GetMetaArchiveChannel())
}

func (a *App) Close() {
	//TODO: error handling
	a.destinationStorage.Close()
	a.backend.Close()
}

func (a *App) Run() {
	defer a.cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := a.uploader.Run(a.ctx)
		if err != nil && err != context.Canceled {
			logger.Errorf("Error executing Uploader: %v", err)
			a.cancel()
			return
		}
	}()

	go func() {
		defer wg.Done()

		err := a.exportManager.Run(a.ctx, a.config.StartLedger, a.config.EndLedger)
		if err != nil {
			logger.Errorf("Error executing ExportManager: %v", err)
			a.cancel()
			return
		}
	}()

	go func() {
		// Handle OS signals to gracefully terminate the service
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-a.ctx.Done():
				logger.Infof("Received context done signal")
				return
			case sig := <-sigCh:
				logger.Infof("Received signal: %v", sig)
				a.cancel()
				return
			}
		}
	}()

	wg.Wait()

	a.Close()

	logger.Info("Shutting down ledger-exporter.")
}

func NewDestinationStorage(ctx context.Context, config *Config) DataStore {
	destinationStorage, err := NewDataStore(ctx, fmt.Sprintf("%s/%s", config.DestinationURL, config.Network))
	logFatalIf(err, "Could not connect to destination storage")
	return destinationStorage
}

// NewLedgerBackend Creates and initializes captive core ledger backend
// Currently, only supports captive-core as ledger backend
func NewLedgerBackend(ctx context.Context, config Config) ledgerbackend.LedgerBackend {
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
