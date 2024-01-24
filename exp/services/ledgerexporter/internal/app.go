package exporter

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/stellar/go/ingest/ledgerbackend"
	_ "github.com/stellar/go/network"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/storage"
)

var (
	logger = supportlog.New().WithField("service", "ledger-exporter")
)

type App struct {
	ctx                context.Context
	cancel             func()
	config             Config
	backend            ledgerbackend.LedgerBackend
	destinationStorage storage.Storage
	exportManager      ExportManager
	uploader           Uploader
}

func NewApp() *App {
	logger.SetLevel(supportlog.InfoLevel)

	config := Config{}
	err := LoadConfig(&config)
	if err != nil {
		logFatalIf(err, "Could not load configuration")
	}
	destinationStorage := NewDestinationStorage(&config)
	backend := NewLedgerBackend(config)

	exportManager := NewExportManager(config.ExporterConfig, backend)

	uploader := NewUploader(destinationStorage, exportManager.GetExportObjectsChannel())

	return &App{
		config:             config,
		backend:            backend,
		destinationStorage: destinationStorage,
		exportManager:      exportManager,
		uploader:           uploader,
	}
}

func (a *App) Close() {
	//TODO: error handling
	a.destinationStorage.Close()
	a.backend.Close()
}

func (a *App) Run() {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	defer a.cancel()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		err := a.uploader.Run(a.ctx)
		if err != nil && err != context.Canceled {
			logger.Errorf("Error executing uploader: %v", err)
			return
		}
	}()

	doneCh := make(chan struct{})
	go func() {
		defer wg.Done()
		defer close(doneCh)

		err := a.exportManager.Run(a.ctx, a.config.StartLedger, a.config.EndLedger)
		if err != nil {
			logger.Errorf("Error executing ExportManager: %v", err)
			return
		}
	}()

	go func() {
		wg.Done()

		// Handle OS signals to gracefully terminate the service
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-doneCh:
				a.cancel()
				return
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

	logger.Info("Shutting down ledgerexporter..")
}

func NewDestinationStorage(config *Config) storage.Storage {
	destinationStorage, err := storage.ConnectBackend(config.DestinationURL, storage.ConnectOptions{})
	logFatalIf(err, "Could not connect to destination storage")
	return destinationStorage
}

// NewLedgerBackend Creates and initializes captive core ledger backend
// Only supports captive core for now
func NewLedgerBackend(config Config) ledgerbackend.LedgerBackend {
	coreConfig := config.StellarCoreConfig
	params := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  coreConfig.NetworkPassphrase,
		HistoryArchiveURLs: coreConfig.HistoryArchiveUrls,
		UseDB:              coreConfig.CaptiveCoreUseDB,
	}
	if coreConfig.CaptiveCoreTomlPath == "" {
		logger.Fatal("Missing captive_core_toml_path in the config")
	}

	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromFile(coreConfig.CaptiveCoreTomlPath, params)
	logFatalIf(err, "Could not create captive core toml")

	captiveConfig := ledgerbackend.CaptiveCoreConfig{
		BinaryPath:          coreConfig.StellarCoreBinaryPath,
		NetworkPassphrase:   params.NetworkPassphrase,
		HistoryArchiveURLs:  params.HistoryArchiveURLs,
		CheckpointFrequency: 64,
		Log:                 logger.WithField("subservice", "stellar-core"),
		Toml:                captiveCoreToml,
		UseDB:               coreConfig.CaptiveCoreUseDB,
	}

	// Create a new captive core backend
	backend, err := ledgerbackend.NewCaptive(captiveConfig)
	logFatalIf(err, "Could not create captive core instance")

	var ledgerRange ledgerbackend.Range
	if config.EndLedger == 0 {
		ledgerRange = ledgerbackend.UnboundedRange(config.StartLedger)
	} else {
		ledgerRange = ledgerbackend.BoundedRange(config.StartLedger, config.EndLedger)
	}

	err = backend.PrepareRange(context.Background(), ledgerRange)
	logFatalIf(err, "Could not prepare captive core ledger backend")
	return backend
}

func logFatalIf(err error, message string, args ...interface{}) {
	if err != nil {
		logger.WithError(err).Fatalf(message, args...)
	}
}
